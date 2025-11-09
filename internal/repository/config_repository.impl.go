package repository

import (
	"time"

	"github.com/LittleAksMax/bids-service/internal/domain"
)

// InMemoryConfigRepository is an in-memory implementation of ConfigurationRepository
type InMemoryConfigRepository struct {
	configs map[string][]*domain.ScheduleConfiguration
}

// NewInMemoryConfigRepository creates a new in-memory repository
func NewInMemoryConfigRepository() *InMemoryConfigRepository {
	return &InMemoryConfigRepository{
		configs: make(map[string][]*domain.ScheduleConfiguration),
	}
}

// GetDue returns configurations that are due for processing
func (r *InMemoryConfigRepository) GetDue() ([]*domain.ScheduleConfiguration, error) {
	currentTime := time.Now().UTC()
	due := make([]*domain.ScheduleConfiguration, 0)
	for _, configsForUser := range r.configs {
		for _, config := range configsForUser {
			if config.DueAt.Before(currentTime) || config.DueAt.Equal(currentTime) {
				due = append(due, config)
			}
		}
	}
	return due, nil
}

func (r *InMemoryConfigRepository) GetByUserID(userID string) ([]*domain.ScheduleConfiguration, bool) {
	configs, ok := r.configs[userID]
	if !ok {
		return nil, false
	}
	return configs, true
}

func (r *InMemoryConfigRepository) Put(config *domain.ScheduleConfiguration) error {
	userConfigs := r.configs[config.UserID] // This works even if key doesn't exist (returns nil)

	// Check if a config with same campaign and marketplace already exists
	for i, existing := range userConfigs { // range over nil slice is safe (skips loop)
		if existing.CampaignID == config.CampaignID && existing.Marketplace == config.Marketplace {
			// Found matching config, so update
			userConfigs[i] = config
			r.configs[config.UserID] = userConfigs // Must assign back to map!
			return nil
		}
	}

	// No matching config found - append new one
	r.configs[config.UserID] = append(userConfigs, config)
	return nil
}
