package processors

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/LittleAksMax/bids-util/retries"
	"github.com/LittleAksMax/bidscript/evaluator"
)

// On failure, we should drive the policy back exactly 1 hour
var failedTimeoutMinutes int64 = 60

const driveScheduleMaxAttempts = 3
const driveScheduleBackoffTime = time.Second * 2

func (p *Processor) handleMessage(ctx context.Context, msg ProcessMessage) {
	state := services.StatePending
	defer func() {
		if p.userService == nil {
			return
		}

		var timeout *int64 = nil
		if state == services.StateFailed {
			timeout = &failedTimeoutMinutes
		}

		p.logger.Infof("Trying to drive schedule")
		err := retries.Retry(driveScheduleMaxAttempts, driveScheduleBackoffTime, p.logger, func(ctx context.Context) error {
			return p.userService.DriveSchedule(ctx, msg.UserID, msg.Profile.ProfileID, state, timeout)
		})(ctx)
		if err != nil {
			p.logger.Errorf("Failed to drive schedule for user %v on profile %d: %v", msg.UserID, msg.Profile.ProfileID, err)
		}
	}()

	p.adsClient.SetRefreshToken(msg.RefreshToken)
	err := p.adsClient.SetRegion(msg.Profile.Region)
	if err != nil {
		_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Failed to set region (possibly invalid)")
		p.logger.Errorf("Failed to set region '%s' (possibly invalid): %v", msg.Profile.Region, err)
		return
	}

	p.logger.Infof("Handling report for user %v on profile %d", msg.UserID, msg.Profile.ProfileID)
	// WithCancelClause is like WithCancel, but we can add a cause (i.e., specific error)
	workCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	reportCh := make(chan reportResult, 1)
	prepCh := make(chan preparationResult, 1)

	// The report and the profile/policy prep can run in parallel, but either side can stop the other
	// by calling cancel() upon encountering an unrecoverable error
	go p.runReportWorker(workCtx, cancel, &msg, reportCh)
	go p.runPreparationWorker(workCtx, cancel, msg.UserID, &msg.Profile, prepCh)

	// We will now wait for both results
	reportRes := <-reportCh
	prepRes := <-prepCh

	if taskIdx, err := selectProcessingError(reportRes.err, prepRes.err); err != nil {
		p.logger.Errorf("failed to process report for user %v on profile %d: %v", msg.UserID, msg.Profile.ProfileID, err)

		if taskIdx == 0 {
			_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Failed to prepare relevant report information for evaluation")
		} else if taskIdx == 1 {
			_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Failed to prepare relevant policy and Ad Group information for evaluation")
		} else {
			// This is just for completeness, but it should never be run
			_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Something went wrong when preparing ")
		}
		state = services.StateFailed
		return
	}

	_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Generated report and fetched policies successfully")

	// Decode generated report
	var adGroupReports []ReportColumns
	if err = reportRes.generatedReport.Decode(&adGroupReports); err != nil {
		p.logger.Errorf("Failed to decode report: %v", err)
		_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Failed to decode generated report")
		state = services.StateFailed
		return
	}

	// Evaluate each adgroup with its policy, and send the results to the consumer
	_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Evaluating policies on Ad Groups")
	for _, adGroupReport := range adGroupReports {
		adGroupID := strconv.FormatInt(adGroupReport.AdGroupID, 10)

		// No entry for this Ad Group (probably no policy attached), just skip it
		adGroupPolicy, ok := prepRes.preparedData.adGroupsByID[adGroupID]
		if !ok {
			continue
		}

		if adGroupPolicy.policy == nil {
			_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, fmt.Sprintf("Failed to evaluate policy (not found) on Ad Group %s", adGroupReport.AdGroupName))
			p.logger.Warnf("[User %s; Profile %d] No prepared policy data found for Ad Group %s, skipping", msg.UserID.String(), msg.Profile.ProfileID, adGroupID)
			state = services.StateErrors
			continue
		}

		p.logger.Infof("[UserID: %s; ProfileID: %d] Evaluating policy %s on AdGroupID: %s", msg.UserID, msg.Profile.ProfileID, adGroupPolicy.policy.ID, adGroupID)

		result, err := p.evaluate(ctx, *adGroupPolicy.policy, adGroupReport)
		if err != nil {
			_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, fmt.Sprintf("Failed to evaluate policy %s on Ad Group %s. Skipping.", adGroupPolicy.policy.Name, adGroupReport.AdGroupName))
			p.logger.Errorf("[UserID: %s; ProfileID: %d] Failed to evaluate policy %s on Ad Group %s: %v", msg.UserID.String(), msg.Profile.ProfileID, adGroupPolicy.policy.Name, adGroupID, err)
			state = services.StateErrors
			continue
		}

		bid := adGroupPolicy.adGroup.Bid
		if bid == nil {
			_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, fmt.Sprintf("No bid information returned for Ad Group %s, skipping policy %s", adGroupReport.AdGroupName, adGroupPolicy.policy.Name))
			p.logger.Errorf("[UserID: %s; ProfileID: %d] No bid information returned for Ad Group %s, skipping policy %s", msg.UserID.String(), msg.Profile.ProfileID, adGroupID, adGroupPolicy.policy.ID)
			state = services.StateErrors
			continue
		}

		defaultBid := bid.DefaultBid
		newBid := defaultBid
		adjustmentAmount := result.Amount
		if result.Percentage {
			adjustmentAmount = defaultBid * (result.Amount / 100.0)
		}
		if result.Operator == evaluator.OperatorEq {
			newBid = result.Amount
		} else if result.Operator == evaluator.OperatorAdd {
			newBid += adjustmentAmount
		} else {
			newBid -= adjustmentAmount
		}

		bidChange := bidChangeResult{
			UserID:     msg.UserID,
			ProfileID:  msg.Profile.ProfileID,
			CampaignID: adGroupPolicy.adGroup.CampaignID,
			AdGroupID:  adGroupID,
			PolicyID:   adGroupPolicy.policy.ID,
			OldBid:     defaultBid,
			NewBid:     newBid,
			IsLive:     adGroupPolicy.isLive,
		}

		if err = p.publishChange(ctx, msg.UserID, &msg.Profile, &bidChange); err != nil {
			_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, fmt.Sprintf("Failed to publish bid change for policy %s on Ad Group %s", adGroupPolicy.policy.Name, adGroupReport.AdGroupName))
			p.logger.Errorf("Failed to publish bid change for policy %s on Ad Group %s: %v", adGroupPolicy.policy.ID, adGroupID, err)
			state = services.StateErrors
			continue
		}
	}
	p.logger.Infof("[UserID: %s; Profile: %d] Done evaluating policies on all Ad Groups", msg.UserID.String(), msg.Profile.ProfileID)
	_, _ = p.userService.Log(ctx, msg.UserID, msg.Profile.ProfileID, "Done evaluating policies on Ad Groups")
}

// selectProcessingError returns the index of the error which was found, and the error itself.
// We use the index to determine which concurrent task failed.
func selectProcessingError(errs ...error) (int, error) {
	var canceledErr error
	// Iterate through the errors received from each worker (can be nil)
	for i, err := range errs {
		if err == nil {
			continue
		}
		if errors.Is(err, context.Canceled) {
			if canceledErr == nil {
				canceledErr = err
			}
			continue
		}

		return i, err
	}

	return -1, canceledErr
}
