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

type preparedProfileData struct {
	adGroupsByID map[string]preparedAdGroup
	// Deprecated: prefer adGroupsByID for ad-group keyed evaluation.
	// attachedByPolicyID map[string][]services.AttachedPolicy
	// Deprecated: prefer adGroupsByID for ad-group keyed evaluation.
	// policiesByID map[string]services.Policy
	// Deprecated: prefer adGroupsByID for ad-group keyed evaluation.
	// adGroupPolicies []adGroupPolicyBinding
}

type preparationResult struct {
	preparedData *preparedProfileData
	err          error
}

type preparedAdGroup struct {
	adGroup adsapimodels.AdGroup
	policy  *services.Policy
	isLive  bool
}

// Deprecated: prefer preparedAdGroup and preparedProfileData.adGroupsByID.
//type adGroupPolicyBinding struct {
//	adGroup adsapimodels.AdGroup
//	policy  services.Policy
//}

func (p *Processor) runPreparationWorker(ctx context.Context, cancel context.CancelCauseFunc, userID uuid.UUID, profile *services.RegionProfile, out chan<- preparationResult) {
	preparedData, err := p.prepareProfileData(ctx, userID, profile)
	if err != nil {
		p.logger.Errorf("[UserID: %s; ProfileID: %d] Failed to prepare profile data: %v", userID.String(), profile.ProfileID, err)
		cancel(fmt.Errorf("profile preparation failed: %w", err))
	}

	out <- preparationResult{
		preparedData: preparedData,
		err:          err,
	}
}

func (p *Processor) prepareProfileData(ctx context.Context, userID uuid.UUID, profile *services.RegionProfile) (*preparedProfileData, error) {
	p.logger.Infof("[UserID: %s; ProfileID: %d] Fetching attached policies", userID.String(), profile.ProfileID)
	attachedPolicies, err := p.fetchAttachedPolicies(ctx, userID, profile.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("fetch attached policies: %w", err)
	}

	// Deprecated: ad-group keyed evaluation no longer needs policy-keyed groupings.
	// p.logger.Infof("Grouping attached policies by ID for user %s on profile %d", userID.String(), profile.ProfileID)
	// attachedByPolicyID := groupAttachedPoliciesByPolicyID(attachedPolicies)

	// No policies in this profile (for some reason), so leave early with no error
	if len(attachedPolicies) == 0 {
		p.logger.Infof("[UserID: %s; ProfileID: %d] No attached policies found, skipping preparation", userID.String(), profile.ProfileID)
		return &preparedProfileData{
			adGroupsByID: map[string]preparedAdGroup{},
			// attachedByPolicyID: attachedByPolicyID,
			// policiesByID:       map[string]services.Policy{},
			// adGroupPolicies:    nil,
		}, nil
	}

	// Get all policies related to profile from policy service
	p.logger.Infof("[UserID: %s; ProfileID: %d] Fetching marketplace policies", userID.String(), profile.ProfileID)
	policies, err := p.fetchPoliciesForProfile(ctx, userID, profile.CountryCode)
	if err != nil {
		return nil, fmt.Errorf("fetch policies: %v", err)
	}

	// Deprecated: ad-group keyed evaluation no longer needs policy-keyed lookups.
	// p.logger.Infof("Building lookup table for attached policies for user %s on profile %d", userID.String(), profile.ProfileID)
	// policiesByID := p.buildAttachedPolicyLookup(attachedByPolicyID, policies)

	// Now, we pair the adgroups with the fetched policies
	p.logger.Infof("[UserID: %s; ProfileID: %d] Fetching Ad Groups for attached policies", userID.String(), profile.ProfileID)
	fetchedAdGroupsByID, err := p.fetchAdGroupsForAttachedPolicies(ctx, profile.ProfileID, attachedPolicies)
	if err != nil {
		return nil, fmt.Errorf("fetch ad groups: %v", err)
	}

	p.logger.Infof("[UserID: %s; ProfileID: %d] Building prepared Ad Group lookup", userID.String(), profile.ProfileID)
	preparedAdGroupsByID := p.buildPreparedAdGroupLookup(userID, profile.ProfileID, attachedPolicies, fetchedAdGroupsByID, policies)

	// Deprecated: ad-group keyed evaluation no longer needs flat ad-group/policy bindings.
	// p.logger.Infof("Joining fetched ad groups to attached policy objects for user %s on profile %d", userID.String(), profile.ProfileID)
	// adGroupPolicies, err := p.buildAdGroupPolicyBindings(attachedPolicies, fetchedAdGroupsByID, policiesByID)
	// if err != nil {
	// 	return nil, err
	// }

	_, _ = p.userService.Log(ctx, userID, profile.ProfileID, "Fetched Ad Groups and attached policies")

	return &preparedProfileData{
		adGroupsByID: preparedAdGroupsByID,
		// attachedByPolicyID: attachedByPolicyID,
		// policiesByID:       policiesByID,
		// adGroupPolicies:    adGroupPolicies,
	}, nil
}

const getAttachedPoliciesMaxAttempts = 3
const getAttachedPoliciesBackoffTime = time.Second * 2

func (p *Processor) fetchAttachedPolicies(ctx context.Context, userID uuid.UUID, profileID int64) ([]services.AttachedPolicy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.userService == nil {
		return nil, fmt.Errorf("user service is not configured")
	}

	var attachedPolicies []services.AttachedPolicy
	err := retries.Retry(getAttachedPoliciesMaxAttempts, getAttachedPoliciesBackoffTime, p.logger, func(ctx context.Context) error {
		policies, err := p.userService.GetAttachedPolicies(ctx, userID, profileID)
		if err != nil {
			return err
		}

		attachedPolicies = policies
		return nil
	})(ctx)
	if err != nil {
		return nil, err
	}

	return attachedPolicies, nil
}

const getProfilePoliciesMaxAttempts = 3
const getProfilePoliciesBackoffTime = time.Second * 2

func (p *Processor) fetchPoliciesForProfile(ctx context.Context, userID uuid.UUID, marketplace string) ([]services.Policy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.policyService == nil {
		return nil, fmt.Errorf("policy service is not configured")
	}

	// In this case, Marketplace and Profile are the same
	var policies []services.Policy
	err := retries.Retry(getProfilePoliciesMaxAttempts, getProfilePoliciesBackoffTime, p.logger, func(ctx context.Context) error {
		fetchedPolicies, err := p.policyService.GetPoliciesForMarketplace(ctx, userID, marketplace)
		if err != nil {
			return err
		}

		policies = fetchedPolicies
		return nil
	})(ctx)
	if err != nil {
		return nil, err
	}

	return policies, nil
}

const getAdGroupsMaxAttempts = 3
const getAdGroupsBackoffTime = time.Second * 2

func (p *Processor) fetchAdGroupsForAttachedPolicies(ctx context.Context, profileID int64, attachedPolicies []services.AttachedPolicy) (map[string]adsapimodels.AdGroup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	adGroupIDs := make([]string, 0, len(attachedPolicies))
	for _, attachedPolicy := range attachedPolicies {
		adGroupIDs = append(adGroupIDs, attachedPolicy.AdGroupID)
	}

	// NOTE: Deleting an AdGroup which has a policy attached will be included in filter
	var adGroups []adsapimodels.AdGroup
	err := retries.Retry(getAdGroupsMaxAttempts, getAdGroupsBackoffTime, p.logger, func(ctx context.Context) error {
		fetchedAdGroups, err := p.adsClient.AdGroupsService.GetAdGroups(profileID, &adsapimodels.ListAdGroupsOptions{
			AdProductFilter: adsapimodels.Filter[adsapimodels.AdProduct]{
				Include: []adsapimodels.AdProduct{adsapimodels.AdProductSP},
			},
			AdGroupIDFilter: &adsapimodels.Filter[string]{
				Include: adGroupIDs,
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
		return nil, fmt.Errorf("fetch ad groups: %v", err)
	}

	adGroupsByID := make(map[string]adsapimodels.AdGroup, len(adGroups))
	for _, adGroup := range adGroups {
		adGroupsByID[adGroup.AdGroupID] = adGroup
	}

	return adGroupsByID, nil
}

func (p *Processor) buildPreparedAdGroupLookup(userID uuid.UUID, profileID int64, attachedPolicies []services.AttachedPolicy, adGroupsByID map[string]adsapimodels.AdGroup, policies []services.Policy) map[string]preparedAdGroup {
	policiesByID := make(map[string]services.Policy, len(policies))
	for _, policy := range policies {
		policiesByID[policy.ID] = policy
	}

	preparedAdGroupsByID := make(map[string]preparedAdGroup, len(adGroupsByID))
	for _, attachedPolicy := range attachedPolicies {
		adGroup, ok := adGroupsByID[attachedPolicy.AdGroupID]
		if !ok {
			continue
		}

		if _, exists := preparedAdGroupsByID[attachedPolicy.AdGroupID]; exists {
			p.logger.Warnf("[UserID: %s; ProfileID: %d; AdGroupID: %s] Multiple attached policies found, keeping the first one", userID.String(), profileID, attachedPolicy.AdGroupID)
			continue
		}

		entry := preparedAdGroup{
			adGroup: adGroup,
			isLive:  attachedPolicy.IsLive,
		}

		policy, ok := policiesByID[attachedPolicy.PolicyID]
		if !ok {
			p.logger.Warnf("[UserID: %s; ProfileID: %d; PolicyID: %s; AdGroupID: %s] Policy was not returned by the policy service, skipping it", userID.String(), profileID, attachedPolicy.PolicyID, attachedPolicy.AdGroupID)
			preparedAdGroupsByID[attachedPolicy.AdGroupID] = entry
			continue
		}

		policyCopy := policy
		entry.policy = &policyCopy
		preparedAdGroupsByID[attachedPolicy.AdGroupID] = entry
	}

	return preparedAdGroupsByID
}

// groupAttachedPoliciesByPolicyID creates a map from the attached policies to make it easier to search
// by policy ID
// Deprecated: prefer buildPreparedAdGroupLookup and preparedProfileData.adGroupsByID.
//func groupAttachedPoliciesByPolicyID(attachedPolicies []services.AttachedPolicy) map[string][]services.AttachedPolicy {
//	grouped := make(map[string][]services.AttachedPolicy)
//	for _, attachedPolicy := range attachedPolicies {
//		grouped[attachedPolicy.PolicyID] = append(grouped[attachedPolicy.PolicyID], attachedPolicy)
//	}
//
//	return grouped
//}

// Deprecated: prefer buildPreparedAdGroupLookup and preparedProfileData.adGroupsByID.
//func (p *Processor) buildAttachedPolicyLookup(attachedByPolicyID map[string][]services.AttachedPolicy, policies []services.Policy) map[string]services.Policy {
//	policiesByID := make(map[string]services.Policy, len(attachedByPolicyID))
//	for _, policy := range policies {
//		if _, needed := attachedByPolicyID[policy.ID]; needed {
//			policiesByID[policy.ID] = policy
//		}
//	}
//
//	// Check which policies were not found
//	for policyID := range attachedByPolicyID {
//		if _, ok := policiesByID[policyID]; !ok {
//			p.logger.Warnf("attached policy %s was not returned by the policy service, skipping it", policyID)
//		}
//	}
//
//	return policiesByID
//}

// Deprecated: prefer buildPreparedAdGroupLookup and preparedProfileData.adGroupsByID.
//func (p *Processor) buildAdGroupPolicyBindings(attachedPolicies []services.AttachedPolicy, adGroupsByID map[string]adsapimodels.AdGroup, policiesByID map[string]services.Policy) ([]adGroupPolicyBinding, error) {
//	bindings := make([]adGroupPolicyBinding, 0, len(attachedPolicies))
//	for _, attachedPolicy := range attachedPolicies {
//		adGroup, ok := adGroupsByID[attachedPolicy.AdGroupID]
//
//		// Ignore adgroups with no attached policy
//		if !ok {
//			continue
//		}
//
//		policy, ok := policiesByID[attachedPolicy.PolicyID]
//
//		// If attached policy not found, skip
//		if !ok {
//			continue
//		}
//
//		bindings = append(bindings, adGroupPolicyBinding{
//			adGroup: adGroup,
//			policy:  policy,
//		})
//	}
//
//	return bindings, nil
//}
