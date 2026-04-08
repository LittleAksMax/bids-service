package services

import (
	"time"

	"github.com/google/uuid"
)

type ScheduleState string

const (
	StatePending    ScheduleState = "PENDING"
	StateFailed     ScheduleState = "FAILED"
	StateProcessing ScheduleState = "PROCESSING"
	StateErrors     ScheduleState = "SOME ERRORS"
)

type processProfilePolicyScheduleRequest struct {
	UserID    uuid.UUID `json:"user_id" validate:"uuid"`
	ProfileID int64     `json:"profile_id" validate:"nonnegative"`
}

type driveProfilePolicyScheduleRequest struct {
	processProfilePolicyScheduleRequest
	TimeoutMinutes *int64 `json:"timeout" validate:"nonnegative"`
}

type ProfilePolicyScheduleResponse struct {
	UserID          string    `json:"user_id"`
	ProfileID       int64     `json:"profile_id"`
	DueAt           time.Time `json:"due_at"`
	IntervalMinutes int64     `json:"interval_minutes"`
}

type AttachedPolicy struct {
	CampaignID string `json:"campaign_id"`
	AdGroupID  string `json:"adgroup_id"`
	PolicyID   string `json:"policy_id"`
	IsLive     bool   `json:"is_live"`
}

// Seller represents a seller with their profiles
type Seller struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Profiles []RegionProfile `json:"profiles"`
}

// RegionProfile represents a profile from a specific region
type RegionProfile struct {
	ProfileID   int64  `json:"profile_id"`
	CountryCode string `json:"country_code"`
	Region      string `json:"region"`
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	AccountType string `json:"account_type"`
}

type CreatedUserLogResponse struct {
	LogID     uuid.UUID `json:"log_id"`
	ProfileID int64     `json:"profile_id"`
	Log       string    `json:"Log"`
	Timestamp time.Time `json:"timestamp"`
}

// BidResponse represents a bid in responses
type BidResponse struct {
	UserID     uuid.UUID `json:"user_id"`
	ProfileID  int64     `json:"profile_id"`
	CampaignID string    `json:"campaign_id"`
	AdGroupID  string    `json:"adgroup_id"`
	PolicyID   string    `json:"policy_id"`
	FromBid    float64   `json:"from_bid"`
	ToBid      float64   `json:"to_bid"`
	ChangeDate time.Time `json:"change_date"`
	IsLive     bool      `json:"is_live"`
}

type UserTokensResponse struct {
	UserID         uuid.UUID `json:"user_id"`
	RefreshTokenEU *string   `json:"refresh_token_eu"`
	RefreshTokenUS *string   `json:"refresh_token_us"`
	RefreshTokenFE *string   `json:"refresh_token_fe"`
}

type createUserLogRequest struct {
	Log string `json:"Log"`
}

// CreateBidRequest represents a request to create a bid
type CreateBidRequest struct {
	ProfileID  int64   `json:"profile_id"`
	CampaignID string  `json:"campaign_id"`
	AdGroupID  string  `json:"adgroup_id"`
	PolicyID   string  `json:"policy_id"`
	FromBid    float64 `json:"from_bid"`
	ToBid      float64 `json:"to_bid"`
	IsLive     bool    `json:"is_live"`
}
