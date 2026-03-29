package processors

import (
	"context"
	"errors"
	"time"

	"github.com/LittleAksMax/bids-service/internal/retries"
	"github.com/LittleAksMax/bids-service/internal/services"
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
			p.logger.Errorf("failed to drive schedule for user %v on profile %d: %v", msg.UserID, msg.Profile.ProfileID, err)
		}
	}()

	if msg.Report == nil {
		p.logger.Errorf("Received nil report object")
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
	go p.runReportWorker(workCtx, cancel, msg.UserID, &msg.Profile, msg.Report, reportCh)
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

	// TODO: evaluate generated report data against the prepared policy/ad group bindings.
	_ = reportRes.generatedReport
	_ = prepRes.preparedData.attachedByPolicyID
	_ = prepRes.preparedData.policiesByID
	_ = prepRes.preparedData.adGroupPolicies
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
