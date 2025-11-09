package repository

import "github.com/LittleAksMax/bids-service/internal/domain"

// ConfigurationRepository defines the interface for configuration data access
type ConfigurationRepository interface {
	// GetDue returns configurations that are due for processing
	GetDue() ([]*domain.ScheduleConfiguration, error)

	// GetByUserID retrieves all configuration for a given UserID.
	// Returns false if no configs for given user found.
	GetByUserID(userID string) ([]*domain.ScheduleConfiguration, bool)

	// Put updates a configuration or creates it if it doesn't exist
	Put(config *domain.ScheduleConfiguration) error
}
