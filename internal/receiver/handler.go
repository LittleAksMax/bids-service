package receiver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LittleAksMax/bids-service/internal/processors"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/LittleAksMax/bids-util/retries"
	"github.com/google/uuid"
)

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
	// Set valid refresh token
	r.adsClient.SetRefreshToken(*refreshToken)

	// Try to set processing on schedule to signal back to users
	r.logger.Infof("Setting schedule to processing state")
	if err = r.userService.SetProcessing(ctx, userID, schedule.ProfileID); err != nil {
		r.logger.Errorf("Failed to mark profile %d as processing: %v", schedule.ProfileID, err)
		return err
	}

	workerIdx := schedule.ProfileID % int64(len(r.workers))
	r.logger.Infof("Dispatching job for user %s on profile %d to Processor %d", schedule.UserID, schedule.ProfileID, workerIdx)
	_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Dispatching job to process")
	r.workers[workerIdx].Notify(processors.ProcessMessage{
		UserID:          userID,
		Profile:         *profile,
		RefreshToken:    *refreshToken,
		DueAt:           schedule.DueAt,
		IntervalMinutes: schedule.IntervalMinutes,
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
