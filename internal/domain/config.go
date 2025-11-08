package domain

import "time"

// ScheduleConfiguration represents a configuration that can be scheduled
type ScheduleConfiguration struct {
	UserID      string
	CampaignID  string
	Marketplace string
	DueAt       time.Time // Should be some multiple of 15 mins
	LastUpdated time.Time
}
