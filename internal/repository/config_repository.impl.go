package repository

import "github.com/LittleAksMax/bids-service/internal/domain"

// InMemoryConfigRepository is an in-memory implementation of ConfigurationRepository
type InMemoryConfigRepository struct {
	configs map[string]*domain.ScheduleConfiguration
}

// NewInMemoryConfigRepository creates a new in-memory repository
func NewInMemoryConfigRepository() *InMemoryConfigRepository {
	return &InMemoryConfigRepository{
		configs: make(map[string]*domain.ScheduleConfiguration),
	}
}

// GetDueConfigurations returns configurations that are due for processing
func (r *InMemoryConfigRepository) GetDueConfigurations() ([]*domain.ScheduleConfiguration, error) {
	// TODO: Implement logic to filter due configurations
	return nil, nil
}

func (r *InMemoryConfigRepository) GetByUserID(userID string) ([]*domain.ScheduleConfiguration, error) {
	// TODO: Implement retrieval logic
	return nil, nil
}

func (r *InMemoryConfigRepository) Put(config *domain.ScheduleConfiguration) error {
	// TODO: Implement update logic
	return nil
}
