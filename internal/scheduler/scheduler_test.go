package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	amznads "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/domain"
)

// Mock repository
type mockConfigRepo struct {
	getDueFunc                   func() ([]*domain.ScheduleConfiguration, error)
	getByUserIDFunc              func(userID string) ([]*domain.ScheduleConfiguration, bool)
	getByUserIDAndCampaignIDFunc func(userID, campaignID string) ([]*domain.ScheduleConfiguration, bool)
	putFunc                      func(config *domain.ScheduleConfiguration) error
	callCount                    int
}

func (m *mockConfigRepo) GetDue() ([]*domain.ScheduleConfiguration, error) {
	m.callCount++
	return m.getDueFunc()
}

func (m *mockConfigRepo) GetByUserID(userID string) ([]*domain.ScheduleConfiguration, bool) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(userID)
	}
	return []*domain.ScheduleConfiguration{}, false
}

func (m *mockConfigRepo) GetByUserIDAndCampaignID(userID, campaignID string) ([]*domain.ScheduleConfiguration, bool) {
	if m.getByUserIDAndCampaignIDFunc != nil {
		return m.getByUserIDAndCampaignIDFunc(userID, campaignID)
	}
	return []*domain.ScheduleConfiguration{}, false
}

func (m *mockConfigRepo) Put(config *domain.ScheduleConfiguration) error {
	if m.putFunc != nil {
		return m.putFunc(config)
	}
	return nil
}

func TestScheduler_poll(t *testing.T) {
	tests := []struct {
		name        string
		getDueFunc  func() ([]*domain.ScheduleConfiguration, error)
		expectError bool
	}{
		{
			name: "successful poll with no due configs",
			getDueFunc: func() ([]*domain.ScheduleConfiguration, error) {
				return []*domain.ScheduleConfiguration{}, nil
			},
			expectError: false,
		},
		{
			name: "repository error",
			getDueFunc: func() ([]*domain.ScheduleConfiguration, error) {
				return nil, errors.New("database error")
			},
			expectError: true,
		},
		{
			name: "successful poll with due configs",
			getDueFunc: func() ([]*domain.ScheduleConfiguration, error) {
				return []*domain.ScheduleConfiguration{
					{
						UserID:      "test_user",
						CampaignID:  "test_campaign",
						Marketplace: "US",
						Interval:    15 * time.Minute,
					},
				}, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockConfigRepo{getDueFunc: tt.getDueFunc}
			s := &Scheduler{
				repo:      mockRepo,
				adsClient: &amznads.AmazonAdsAPIClient{},
			}

			ctx := context.Background()
			s.poll(ctx)

			if mockRepo.callCount != 1 {
				t.Errorf("expected GetDue to be called once, got %d", mockRepo.callCount)
			}
		})
	}
}

func TestScheduler_Start(t *testing.T) {
	t.Run("polls at specified interval", func(t *testing.T) {
		mockRepo := &mockConfigRepo{
			getDueFunc: func() ([]*domain.ScheduleConfiguration, error) {
				return []*domain.ScheduleConfiguration{}, nil
			},
		}

		s := &Scheduler{
			repo:         mockRepo,
			pollInterval: 100 * time.Millisecond,
			adsClient:    &amznads.AmazonAdsAPIClient{},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
		defer cancel()

		s.Start(ctx)

		// Should poll approximately 3 times in 350ms with 100ms interval
		if mockRepo.callCount < 3 || mockRepo.callCount > 4 {
			t.Errorf("expected 3-4 polls, got %d", mockRepo.callCount)
		}
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		mockRepo := &mockConfigRepo{
			getDueFunc: func() ([]*domain.ScheduleConfiguration, error) {
				return []*domain.ScheduleConfiguration{}, nil
			},
		}

		s := &Scheduler{
			repo:         mockRepo,
			pollInterval: 50 * time.Millisecond,
			adsClient:    &amznads.AmazonAdsAPIClient{},
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan bool)
		go func() {
			s.Start(ctx)
			done <- true
		}()

		time.Sleep(125 * time.Millisecond) // Let it poll a few times
		cancel()                           // Cancel the context

		select {
		case <-done:
			// Scheduler stopped successfully
		case <-time.After(1 * time.Second):
			t.Error("scheduler did not stop after context cancellation")
		}

		if mockRepo.callCount < 2 {
			t.Errorf("expected at least 2 polls before cancellation, got %d", mockRepo.callCount)
		}
	})
}
