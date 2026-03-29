package receiver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	adsapimodels "github.com/LittleAksMax/amazon-ads-api-sdk-go/models"
	"github.com/LittleAksMax/bids-service/internal/processors"
	"github.com/LittleAksMax/bids-service/internal/retries"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/google/uuid"
)

const cancelReportAttempts = 3
const cancelReportRetryBackoffTime = time.Second * 2

const getTokensAttempts = 3
const getTokensBackoffTime = time.Second * 2

const getProfilesAttempts = 3
const getProfilesRetryBackoffTime = time.Second * 2

func (r *Receiver) handleSchedule(ctx context.Context, schedule *services.ProfilePolicyScheduleResponse) error {
	userID, err := uuid.Parse(schedule.UserID)
	if err != nil {
		r.logger.Errorf("Failed to parse user ID: %v", err)
		_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Failed to parse user ID")
		return err
	}

	r.logger.Infof("Processing schedule for user %s on profile %d", schedule.UserID, schedule.ProfileID)

	r.logger.Infof("Fetching profile information for user %s on profile %d", schedule.UserID, schedule.ProfileID)
	var profile *services.RegionProfile = nil
	err = retries.Retry(getProfilesAttempts, getProfilesRetryBackoffTime, r.logger, func(ctx context.Context) error {
		p, err := r.profileCache.GetProfile(ctx, userID, schedule.ProfileID)
		if err != nil {
			return err
		}
		profile = p
		return nil
	})(ctx)
	if err != nil {
		r.logger.Errorf("Failed to fetch profile information for user %s on profile %d: %v", schedule.UserID, schedule.ProfileID, err)
		_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Failed to get profile information")
		return err
	}

	r.logger.Infof("Setting client region '%s' for user %s based on fetched profile %d", profile.Region, schedule.UserID, schedule.ProfileID)
	err = r.adsClient.SetRegion(profile.Region)
	if err != nil {
		r.logger.Errorf("Failed to set region '%s' for user %s on profile %d: %v", profile.Region, schedule.UserID, schedule.ProfileID, err)
		return err
	}

	r.logger.Infof("Getting tokens for user %s", schedule.UserID)
	var tokens *services.UserTokensResponse = nil
	err = retries.Retry(getTokensAttempts, getTokensBackoffTime, r.logger, func(ctx context.Context) error {
		t, err := r.userService.GetTokens(ctx, userID)
		tokens = t
		return err
	})(ctx)
	if err != nil {
		r.logger.Errorf("Failed to retrieve user's authentication tokens: %v", err)
		_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Failed to retrieve user authentication tokens")
		return err
	}

	// Select relevant refresh token for Amazon Ads client
	r.logger.Infof("Setting regional refresh token for user %s on profile %d", schedule.UserID, schedule.ProfileID)
	refreshToken, regionOk := getRelevantToken(profile.Region, tokens)
	if refreshToken == nil {
		// If the region is correct, then the refresh token is not in the database for user, might need to do Lwa
		if regionOk {
			_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, fmt.Sprintf("Couldn't find refresh token. Maybe you need to perform LwA on region '%s'", profile.Region))
			r.logger.Errorf("User %s has no defined refresh token on region '%s'", schedule.UserID, profile.Region)
		} else {
			_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, fmt.Sprintf("Invalid profile region '%s'; this is likely our fault.", profile.Region))
			r.logger.Errorf("Invalid region '%s' for user %s on profile %d", profile.Region, schedule.UserID, schedule.ProfileID)
		}
		return errors.New("failed to match refresh token")
	}
	r.adsClient.SetRefreshToken(*refreshToken)

	// Use adsClient to execute the required Ads API request for this schedule
	reportOpts := adsapimodels.NewSponsoredProductsTargetingReport(
		fmt.Sprintf("%s-%d-%s", schedule.UserID, schedule.ProfileID, schedule.DueAt.String()),
		time.Now().UTC().Add(-time.Duration(schedule.IntervalMinutes)*time.Minute),
		time.Now().UTC(),
		[]string{
			"campaignName",
			"campaignId",
			"adGroupName",
			"adGroupId",
			"impressions",
			"clicks",
			"cost",
			"sales1d",
		},
	)

	// Request report generation
	r.logger.Infof("Requesting report for user %s on profile %d", schedule.UserID, schedule.ProfileID)
	report, err := r.adsClient.ReportsService.RequestReport(ctx, schedule.ProfileID, reportOpts)
	if err != nil {
		r.logger.Errorf("Failed to request report for user %s on profile %d: %v", schedule.UserID, schedule.ProfileID, err)
		_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Failed to request report from Amazon")
		return err
	}

	// Try to set processing on schedule to signal back to users
	r.logger.Infof("Setting schedule to processing state")
	if err = r.userService.SetProcessing(ctx, userID, schedule.ProfileID); err != nil {
		r.logger.Errorf("Failed to mark profile %d as processing: %v. Cancelling report %s.", schedule.ProfileID, err, report.ReportID())
		reportCancelErr := retries.Retry(cancelReportAttempts, cancelReportRetryBackoffTime, r.logger, func(ctx context.Context) error {
			return r.adsClient.ReportsService.CancelReport(ctx, schedule.ProfileID, report.ReportID())
		})(ctx)
		if reportCancelErr != nil {
			_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Failed to cancel report")
			r.logger.Errorf("Failed to cancel report %s on for user %s profile %d: %v.", report.ReportID(), schedule.UserID, schedule.ProfileID, err)
		}
		return errors.Join(err, reportCancelErr)
	}

	workerIdx := schedule.ProfileID % int64(len(r.workers))
	r.logger.Infof("Dispatching report for user %s on profile %d to Processor %d", schedule.UserID, schedule.ProfileID, workerIdx)
	_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, fmt.Sprintf("Dispatching report (Seller: %s; Profile Marketplace: %s)", profile.AccountName, profile.CountryCode))
	r.workers[workerIdx].Notify(processors.ProcessMessage{
		UserID:  userID,
		Profile: *profile,
		Report:  report,
	})

	return nil
}

// getRelevantToken for a profile's region given the user's tokens.
// The bool output is true if the profile region is valid (e.g., EU, US, FE).
func getRelevantToken(profileRegion string, tokens *services.UserTokensResponse) (*string, bool) {
	normalisedRegion := strings.ToUpper(profileRegion) // just in case
	switch normalisedRegion {
	case "EU":
		return tokens.RefreshTokenEU, true
	case "US":
		return tokens.RefreshTokenUS, true
	case "FE":
		return tokens.RefreshTokenFE, true
	default:
		return nil, false
	}
}
