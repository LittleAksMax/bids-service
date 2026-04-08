package processors

import (
	"context"
	"fmt"
	"time"

	adsapimodels "github.com/LittleAksMax/amazon-ads-api-sdk-go/models"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/LittleAksMax/bids-util/retries"
	"github.com/google/uuid"
)

const writeChangeLogMaxAttempts = 3
const writeChangeLogBackoffTime = time.Second * 2

const changeBidMaxAttempts = 3
const changeBidMaxBackoffTime = time.Second * 2

func (p *Processor) publishChange(ctx context.Context, userID uuid.UUID, profile *services.RegionProfile, change *bidChangeResult) error {
	p.logger.Infof("[UserID: %s; ProfileID: %d; AdGroupID: %s] Publishing bid change", userID.String(), profile.ProfileID, change.AdGroupID)

	adGroupName := ""
	if change.IsLive {
		// No point making a request if no change
		if change.OldBid == change.NewBid {
			p.logger.Infof("[UserID: %s; ProfileID: %d; AdGroupID: %s] No bid change needed (current bid %.4f)", userID.String(), profile.ProfileID, change.AdGroupID, change.OldBid)
			return nil
		}
		adGroup, err := p.fetchAdGroupForChange(ctx, profile.ProfileID, change.AdGroupID)
		if err != nil {
			_, _ = p.userService.Log(ctx, userID, profile.ProfileID, fmt.Sprintf("Couldn't fetch Ad Group"))
			return fmt.Errorf("fetch ad group %s: %w", change.AdGroupID, err)
		}

		if err := p.updateAdGroupBid(ctx, profile.ProfileID, adGroup, change.NewBid); err != nil {
			_, _ = p.userService.Log(ctx, userID, profile.ProfileID, fmt.Sprintf("Couldn't update Ad Group %s", adGroup.Name))
			return fmt.Errorf("update ad group %s: %w", change.AdGroupID, err)
		}

		targets, err := p.fetchTargetsForChange(ctx, profile.ProfileID, change.AdGroupID)
		if err != nil {
			_, _ = p.userService.Log(ctx, userID, profile.ProfileID, fmt.Sprintf("Couldn't fetch Targets for Ad Group %s", adGroup.Name))
			return fmt.Errorf("fetch targets for ad group %s: %w", change.AdGroupID, err)
		}

		if err := p.updateTargetBids(ctx, profile.ProfileID, targets, change.NewBid); err != nil {
			_, _ = p.userService.Log(ctx, userID, profile.ProfileID, fmt.Sprintf("Couldn't update targets for Ad Group %s", adGroup.Name))
			p.logger.Errorf("[UserID: %s; ProfileID: %d; AdGroupID: %s] Failed to update targets", userID.String(), profile.ProfileID, change.AdGroupID)
			return err
		}
	}

	if err := p.createBidChange(ctx, userID, profile, change); err != nil {
		_, _ = p.userService.Log(ctx, userID, profile.ProfileID, fmt.Sprintf("Published bid change to Amazon, failed to update change in platform database for Ad Group %s", adGroupName))
		return fmt.Errorf("create bid record for ad group %s: %w", change.AdGroupID, err)
	}

	return nil
}

func (p *Processor) createBidChange(ctx context.Context, userID uuid.UUID, profile *services.RegionProfile, change *bidChangeResult) error {
	return retries.Retry(writeChangeLogMaxAttempts, writeChangeLogBackoffTime, p.logger, func(ctx context.Context) error {
		_, err := p.userService.CreateBid(ctx, userID, services.CreateBidRequest{
			ProfileID:  profile.ProfileID,
			CampaignID: change.CampaignID,
			AdGroupID:  change.AdGroupID,
			PolicyID:   change.PolicyID,
			FromBid:    change.OldBid,
			ToBid:      change.NewBid,
			IsLive:     change.IsLive,
		})
		return err
	})(ctx)
}

func (p *Processor) fetchAdGroupForChange(ctx context.Context, profileID int64, adGroupID string) (*adsapimodels.AdGroup, error) {
	var adGroups []adsapimodels.AdGroup
	err := retries.Retry(changeBidMaxAttempts, changeBidMaxBackoffTime, p.logger, func(ctx context.Context) error {
		fetchedAdGroups, err := p.adsClient.AdGroupsService.GetAdGroups(profileID, &adsapimodels.ListAdGroupsOptions{
			AdProductFilter: adsapimodels.Filter[adsapimodels.AdProduct]{
				Include: []adsapimodels.AdProduct{adsapimodels.AdProductSP},
			},
			AdGroupIDFilter: &adsapimodels.Filter[string]{
				Include: []string{adGroupID},
			},
			StateFilter: &adsapimodels.Filter[adsapimodels.State]{
				Include: []adsapimodels.State{adsapimodels.StateEnabled},
			},
		}).Collect(ctx)
		if err != nil {
			return err
		}

		adGroups = fetchedAdGroups
		return nil
	})(ctx)
	if err != nil {
		return nil, err
	}
	if len(adGroups) != 1 {
		return nil, fmt.Errorf("expected 1 ad group, got %d", len(adGroups))
	}

	return &adGroups[0], nil
}

func (p *Processor) updateAdGroupBid(ctx context.Context, profileID int64, adGroup *adsapimodels.AdGroup, newBid float64) error {
	if adGroup == nil {
		return fmt.Errorf("ad group is nil")
	}
	if adGroup.Bid == nil {
		return fmt.Errorf("ad group %s has no bid", adGroup.AdGroupID)
	}

	bid := *adGroup.Bid
	bid.DefaultBid = newBid

	var response *adsapimodels.UpdateAdGroupsResponse
	err := retries.Retry(changeBidMaxAttempts, changeBidMaxBackoffTime, p.logger, func(ctx context.Context) error {
		updatedAdGroups, err := p.adsClient.AdGroupsService.UpdateAdGroups(ctx, profileID, &adsapimodels.UpdateAdGroupsOptions{
			AdGroups: []adsapimodels.UpdateAdGroupOption{
				{
					AdGroupID: adGroup.AdGroupID,
					Bid:       &bid,
				},
			},
		})
		if err != nil {
			return err
		}

		response = updatedAdGroups
		return nil
	})(ctx)
	if err != nil {
		return err
	}
	if response == nil || len(response.Success) != 1 {
		return fmt.Errorf("unexpected ad group update response")
	}
	if response.Success[0].AdGroup.Bid == nil || response.Success[0].AdGroup.Bid.DefaultBid != newBid {
		return fmt.Errorf("ad group %s was not updated to bid %.2f", adGroup.AdGroupID, newBid)
	}

	return nil
}

func (p *Processor) fetchTargetsForChange(ctx context.Context, profileID int64, adGroupID string) ([]adsapimodels.Target, error) {
	var targets []adsapimodels.Target
	err := retries.Retry(changeBidMaxAttempts, changeBidMaxBackoffTime, p.logger, func(ctx context.Context) error {
		fetchedTargets, err := p.adsClient.TargetsService.GetTargets(profileID, &adsapimodels.ListTargetsOptions{
			AdProductFilter: adsapimodels.Filter[adsapimodels.AdProduct]{
				Include: []adsapimodels.AdProduct{adsapimodels.AdProductSP},
			},
			AdGroupIDFilter: &adsapimodels.Filter[string]{
				Include: []string{adGroupID},
			},
			StateFilter: &adsapimodels.Filter[adsapimodels.State]{
				Include: []adsapimodels.State{adsapimodels.StateEnabled},
			},
		}).Collect(ctx)
		if err != nil {
			return err
		}

		targets = fetchedTargets
		return nil
	})(ctx)
	if err != nil {
		return nil, err
	}

	return targets, nil
}

func (p *Processor) updateTargetBids(ctx context.Context, profileID int64, targets []adsapimodels.Target, newBid float64) error {
	// Fetch targets to change (there may be none to change, or they might have already changed from just
	// setting the Ad Group's default bid)
	updates := buildTargetBidUpdates(targets, newBid)
	if len(updates) == 0 {
		return nil
	}

	var response *adsapimodels.UpdateTargetsResponse
	err := retries.Retry(changeBidMaxAttempts, changeBidMaxBackoffTime, p.logger, func(ctx context.Context) error {
		updatedTargets, err := p.adsClient.TargetsService.UpdateTargets(ctx, profileID, &adsapimodels.UpdateTargetsOptions{
			Targets: updates,
		})
		if err != nil {
			return err
		}

		response = updatedTargets
		return nil
	})(ctx)
	if err != nil {
		return err
	}
	successCount := 0
	if response != nil {
		successCount = len(response.Success)
	}
	if response == nil || successCount != len(updates) {
		return fmt.Errorf("updated %d of %d targets", successCount, len(updates))
	}

	for _, target := range response.Success {
		if target.Target.Bid == nil || target.Target.Bid.Bid != newBid {
			return fmt.Errorf("target %s was not updated to bid %.2f", target.Target.TargetID, newBid)
		}
	}

	return nil
}

func buildTargetBidUpdates(targets []adsapimodels.Target, newBid float64) []adsapimodels.UpdateTargetOption {
	updates := make([]adsapimodels.UpdateTargetOption, 0, len(targets))
	for _, target := range targets {
		// This may be true for older Ad Groups, where the different target types
		// became available to manage for the different
		if target.Bid == nil {
			continue
		}

		// Only set the target to be updated if there is a difference in the bid values
		// This is because we can't be sure that adjusting the default bid of the Ad Group
		// automatically applies the change to targets, due to older versions of targets
		// still being in use
		bid := *target.Bid
		if bid.Bid != newBid {
			bid.Bid = newBid
			updates = append(updates, adsapimodels.UpdateTargetOption{
				TargetID: target.TargetID,
				Bid:      &bid,
			})
		}
	}

	return updates
}
