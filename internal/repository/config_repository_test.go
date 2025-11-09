package repository

import (
	"testing"
	"time"

	"github.com/LittleAksMax/bids-service/internal/domain"
)

func TestInMemoryConfigRepository_Put(t *testing.T) {
	tests := []struct {
		name            string
		existingConfig  *domain.ScheduleConfiguration
		newConfig       *domain.ScheduleConfiguration
		shouldOverwrite bool
	}{
		{
			name: "insert new config",
			newConfig: &domain.ScheduleConfiguration{
				UserID:      "user1",
				CampaignID:  "campaign1",
				Marketplace: "US",
				Interval:    30 * time.Minute,
			},
			shouldOverwrite: false,
		},
		{
			name: "update existing config with same campaign and marketplace",
			existingConfig: &domain.ScheduleConfiguration{
				UserID:      "user1",
				CampaignID:  "campaign1",
				Marketplace: "US",
				Interval:    30 * time.Minute,
			},
			newConfig: &domain.ScheduleConfiguration{
				UserID:      "user1",
				CampaignID:  "campaign1",
				Marketplace: "US",
				Interval:    60 * time.Minute,
			},
			shouldOverwrite: true,
		},
		{
			name: "insert new config with different marketplace",
			existingConfig: &domain.ScheduleConfiguration{
				UserID:      "user1",
				CampaignID:  "campaign1",
				Marketplace: "US",
				Interval:    30 * time.Minute,
			},
			newConfig: &domain.ScheduleConfiguration{
				UserID:      "user1",
				CampaignID:  "campaign1",
				Marketplace: "UK",
				Interval:    45 * time.Minute,
			},
			shouldOverwrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryConfigRepository()

			if tt.existingConfig != nil {
				err := repo.Put(tt.existingConfig)
				if err != nil {
					t.Fatalf("failed to insert existing config: %v", err)
				}
			}

			err := repo.Put(tt.newConfig)
			if err != nil {
				t.Fatalf("failed to put new config: %v", err)
			}

			// Verify the config was stored
			configs, found := repo.GetByUserID(tt.newConfig.UserID)
			if !found {
				t.Fatal("expected to find configs for user")
			}

			if tt.shouldOverwrite {
				if len(configs) != 1 {
					t.Errorf("expected 1 config after overwrite, got %d", len(configs))
				}
				if configs[0].Interval != tt.newConfig.Interval {
					t.Errorf("expected interval %v, got %v", tt.newConfig.Interval, configs[0].Interval)
				}
			} else {
				expectedCount := 1
				if tt.existingConfig != nil {
					expectedCount = 2
				}
				if len(configs) != expectedCount {
					t.Errorf("expected %d configs, got %d", expectedCount, len(configs))
				}
			}
		})
	}
}

func TestInMemoryConfigRepository_GetByUserID(t *testing.T) {
	repo := NewInMemoryConfigRepository()

	// Add test data
	config1 := &domain.ScheduleConfiguration{
		UserID:      "user1",
		CampaignID:  "campaign1",
		Marketplace: "US",
		Interval:    30 * time.Minute,
	}
	config2 := &domain.ScheduleConfiguration{
		UserID:      "user1",
		CampaignID:  "campaign2",
		Marketplace: "UK",
		Interval:    45 * time.Minute,
	}
	config3 := &domain.ScheduleConfiguration{
		UserID:      "user2",
		CampaignID:  "campaign3",
		Marketplace: "US",
		Interval:    60 * time.Minute,
	}

	repo.Put(config1)
	repo.Put(config2)
	repo.Put(config3)

	tests := []struct {
		name          string
		userID        string
		expectFound   bool
		expectedCount int
	}{
		{
			name:          "user with multiple configs",
			userID:        "user1",
			expectFound:   true,
			expectedCount: 2,
		},
		{
			name:          "user with single config",
			userID:        "user2",
			expectFound:   true,
			expectedCount: 1,
		},
		{
			name:          "non-existent user",
			userID:        "user999",
			expectFound:   false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs, found := repo.GetByUserID(tt.userID)

			if found != tt.expectFound {
				t.Errorf("expected found=%v, got %v", tt.expectFound, found)
			}

			if found && len(configs) != tt.expectedCount {
				t.Errorf("expected %d configs, got %d", tt.expectedCount, len(configs))
			}
		})
	}
}

func TestInMemoryConfigRepository_GetByUserIDAndCampaignID(t *testing.T) {
	repo := NewInMemoryConfigRepository()

	// Add test data
	config1 := &domain.ScheduleConfiguration{
		UserID:      "user1",
		CampaignID:  "campaign1",
		Marketplace: "US",
		Interval:    30 * time.Minute,
	}
	config2 := &domain.ScheduleConfiguration{
		UserID:      "user1",
		CampaignID:  "campaign1",
		Marketplace: "UK",
		Interval:    45 * time.Minute,
	}
	config3 := &domain.ScheduleConfiguration{
		UserID:      "user1",
		CampaignID:  "campaign2",
		Marketplace: "US",
		Interval:    60 * time.Minute,
	}

	repo.Put(config1)
	repo.Put(config2)
	repo.Put(config3)

	tests := []struct {
		name          string
		userID        string
		campaignID    string
		expectFound   bool
		expectedCount int
	}{
		{
			name:          "campaign with multiple marketplaces",
			userID:        "user1",
			campaignID:    "campaign1",
			expectFound:   true,
			expectedCount: 2,
		},
		{
			name:          "campaign with single config",
			userID:        "user1",
			campaignID:    "campaign2",
			expectFound:   true,
			expectedCount: 1,
		},
		{
			name:          "non-existent campaign",
			userID:        "user1",
			campaignID:    "campaign999",
			expectFound:   false,
			expectedCount: 0,
		},
		{
			name:          "non-existent user",
			userID:        "user999",
			campaignID:    "campaign1",
			expectFound:   false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configs, found := repo.GetByUserIDAndCampaignID(tt.userID, tt.campaignID)

			if found != tt.expectFound {
				t.Errorf("expected found=%v, got %v", tt.expectFound, found)
			}

			if found && len(configs) != tt.expectedCount {
				t.Errorf("expected %d configs, got %d", tt.expectedCount, len(configs))
			}

			// Verify all returned configs have the correct campaign ID
			if found {
				for _, config := range configs {
					if config.CampaignID != tt.campaignID {
						t.Errorf("expected campaignID %q, got %q", tt.campaignID, config.CampaignID)
					}
				}
			}
		})
	}
}

func TestInMemoryConfigRepository_GetDue(t *testing.T) {
	repo := NewInMemoryConfigRepository()

	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Add test data
	pastDue := &domain.ScheduleConfiguration{
		UserID:      "user1",
		CampaignID:  "campaign1",
		Marketplace: "US",
		DueAt:       past,
		Interval:    30 * time.Minute,
	}
	currentlyDue := &domain.ScheduleConfiguration{
		UserID:      "user1",
		CampaignID:  "campaign2",
		Marketplace: "UK",
		DueAt:       now,
		Interval:    45 * time.Minute,
	}
	notYetDue := &domain.ScheduleConfiguration{
		UserID:      "user2",
		CampaignID:  "campaign3",
		Marketplace: "US",
		DueAt:       future,
		Interval:    60 * time.Minute,
	}

	repo.Put(pastDue)
	repo.Put(currentlyDue)
	repo.Put(notYetDue)

	dueConfigs, err := repo.GetDue()
	if err != nil {
		t.Fatalf("GetDue returned error: %v", err)
	}

	// Should return 2 configs (past and current, but not future)
	if len(dueConfigs) != 2 {
		t.Errorf("expected 2 due configs, got %d", len(dueConfigs))
	}

	// Verify none of the due configs are in the future
	for _, config := range dueConfigs {
		if config.DueAt.After(now) {
			t.Errorf("found config with future DueAt: %v", config.DueAt)
		}
	}
}

func TestInMemoryConfigRepository_PutNilSlice(t *testing.T) {
	repo := NewInMemoryConfigRepository()

	// Insert config for a user that doesn't exist yet
	config := &domain.ScheduleConfiguration{
		UserID:      "newuser",
		CampaignID:  "campaign1",
		Marketplace: "US",
		Interval:    30 * time.Minute,
	}

	err := repo.Put(config)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify it was added
	configs, found := repo.GetByUserID("newuser")
	if !found {
		t.Fatal("expected to find configs for new user")
	}

	if len(configs) != 1 {
		t.Errorf("expected 1 config, got %d", len(configs))
	}
}
