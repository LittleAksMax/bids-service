package processors

import "github.com/google/uuid"

// https://advertising.amazon.com/API/docs/en-us/guides/reporting/v3/columns
// We want columns for 'spCampaigns' type reports
var reportColumns = []string{
	"campaignName",
	"campaignId",
	"adGroupName",
	"adGroupId",
	"impressions",
	"clicks",
	"cost",
	"purchases1d", // "purchases within 1 day of click"
	"sales1d",     // "sales within 1 day of click"
	"clickThroughRate",
	"costPerClick",
}

// NOTE: reporting API gives IDs as large ints, rather than strings?
type ReportColumns struct {
	CampaignName     string  `json:"campaignName"`
	CampaignID       int64   `json:"campaignId"`
	AdGroupName      string  `json:"adGroupName"`
	AdGroupID        int64   `json:"adGroupId"`
	Impressions      int     `json:"impressions"`
	Clicks           int     `json:"clicks"`
	Cost             float64 `json:"cost"`
	Purchases1d      int     `json:"purchases1d"`
	Sales1d          float64 `json:"sales1d"`
	ClickThroughRate float64 `json:"clickThroughRate"`
	CostPerClick     float64 `json:"costPerClick"`
}

type bidChangeResult struct {
	UserID     uuid.UUID `json:"user_id"`
	ProfileID  int64     `json:"profile_id"`
	CampaignID string    `json:"campaign_id"`
	AdGroupID  string    `json:"adgroup_id"`
	PolicyID   string    `json:"policy_id"`
	OldBid     float64   `json:"old_bid"`
	NewBid     float64   `json:"new_bid"`
	IsLive     bool      `json:"is_live"`
}
