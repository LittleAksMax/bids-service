package handler

import (
	"time"

	"github.com/LittleAksMax/bids-service/internal/domain"
)

// ScheduleConfigRequest represents the JSON request body for creating/updating a schedule configuration
type ScheduleConfigRequest struct {
	UserID      string `json:"userId"`
	CampaignID  string `json:"campaignId"`
	Marketplace string `json:"marketplace"`
	Interval    int    `json:"interval"` // Interval in minutes (must be a multiple of 15)
}

// Validate checks if all required fields are present and valid
func (r *ScheduleConfigRequest) Validate() map[string]string {
	errors := make(map[string]string)

	if r.UserID == "" {
		errors["userId"] = "userId is required"
	}

	if r.CampaignID == "" {
		errors["campaignId"] = "campaignId is required"
	}

	if r.Marketplace == "" {
		errors["marketplace"] = "marketplace is required"
	}

	if r.Interval <= 0 {
		errors["interval"] = "interval is required and must be greater than 0"
	}

	// Validate that interval is a multiple of 15 minutes
	if r.Interval > 0 && r.Interval%15 != 0 {
		errors["interval"] = "interval must be a multiple of 15 minutes (e.g., 15, 30, 45, 60, etc.)"
	}

	return errors
}

// ToDomain converts the request DTO to a domain entity
func (r *ScheduleConfigRequest) ToDomain() *domain.ScheduleConfiguration {
	// Calculate DueAt by adding the interval to the current time
	now := time.Now()
	dueAt := now.Add(time.Duration(r.Interval) * time.Minute)

	return &domain.ScheduleConfiguration{
		UserID:      r.UserID,
		CampaignID:  r.CampaignID,
		Marketplace: r.Marketplace,
		DueAt:       dueAt,
		LastUpdated: now,
	}
}
