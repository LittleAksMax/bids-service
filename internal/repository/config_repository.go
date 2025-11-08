package repository

import "github.com/LittleAksMax/bids-service/internal/domain"

// ConfigurationRepository defines the interface for configuration data access
type ConfigurationRepository interface {
	// GetDueConfigurations returns configurations that are due for processing
	GetDueConfigurations() ([]*domain.ScheduleConfiguration, error)

	// GetByUserID retrieves all configuration for a given UserID
	GetByUserID(userID string) ([]*domain.ScheduleConfiguration, error)

	// Put updates a configuration or creates it if it doesn't exist
	Put(config *domain.ScheduleConfiguration) error
}
