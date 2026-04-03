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
		r.logger.Errorf("[UserID: %s; ProfileID: %d] Failed to parse user ID: %v", schedule.UserID, schedule.ProfileID, err)
		_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Failed to parse user ID")
		return err
	}

	r.logger.Infof("[UserID: %s; ProfileID: %d] Processing schedule", schedule.UserID, schedule.ProfileID)

	r.logger.Infof("[UserID: %s; ProfileID: %d] Fetching profile information", schedule.UserID, schedule.ProfileID)
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
		r.logger.Errorf("[UserID: %s; ProfileID: %d] Failed to fetch profile information: %v", schedule.UserID, schedule.ProfileID, err)
		_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Failed to get profile information")
		return err
	}

	r.logger.Infof("[UserID: %s; ProfileID: %d] Fetching user tokens", schedule.UserID, schedule.ProfileID)
	var tokens *services.UserTokensResponse = nil
	err = retries.Retry(getTokensAttempts, getTokensBackoffTime, r.logger, func(ctx context.Context) error {
		t, err := r.userService.GetTokens(ctx, userID)
		tokens = t
		return err
	})(ctx)
	if err != nil {
		r.logger.Errorf("[UserID: %s; ProfileID: %d] Failed to retrieve user authentication tokens: %v", schedule.UserID, schedule.ProfileID, err)
		_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, "Failed to retrieve user authentication tokens")
		return err
	}

	// Select relevant refresh token for Amazon Ads client
	r.logger.Infof("[UserID: %s; ProfileID: %d] Selecting regional refresh token", schedule.UserID, schedule.ProfileID)
	refreshToken, regionOk := getRelevantToken(profile.Region, tokens)
	if refreshToken == nil {
		// If the region is correct, then the refresh token is not in the database for user, might need to do Lwa
		if regionOk {
			_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, fmt.Sprintf("Couldn't find refresh token. Maybe you need to perform LwA on region '%s'", profile.Region))
			r.logger.Errorf("[UserID: %s; ProfileID: %d] No refresh token defined for region '%s'", schedule.UserID, schedule.ProfileID, profile.Region)
		} else {
			_, _ = r.userService.Log(ctx, userID, schedule.ProfileID, fmt.Sprintf("Invalid profile region '%s'; this is likely our fault.", profile.Region))
			r.logger.Errorf("[UserID: %s; ProfileID: %d] Invalid profile region '%s'", schedule.UserID, schedule.ProfileID, profile.Region)
		}
		return errors.New("failed to match refresh token")
	}

	// Try to set processing on schedule to signal back to users
	r.logger.Infof("[UserID: %s; ProfileID: %d] Marking schedule as processing", schedule.UserID, schedule.ProfileID)
	if err = r.userService.SetProcessing(ctx, userID, schedule.ProfileID); err != nil {
		r.logger.Errorf("[UserID: %s; ProfileID: %d] Failed to mark schedule as processing: %v", schedule.UserID, schedule.ProfileID, err)
		return err
	}

	workerIdx := schedule.ProfileID % int64(len(r.workers))
	r.logger.Infof("[UserID: %s; ProfileID: %d] Dispatching job to Processor %d", schedule.UserID, schedule.ProfileID, workerIdx)
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
