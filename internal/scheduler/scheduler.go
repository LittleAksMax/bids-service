package scheduler

import (
	"context"
	"log"
	"time"

	amznads "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/repository"
)

// Config holds configuration for the scheduler
type Config struct {
	ScheduleConfigRepo repository.ConfigurationRepository
	PollInterval       time.Duration
	AdsClient          *amznads.AmazonAdsAPIClient
}

// Scheduler handles periodic polling and processing of due configurations
type Scheduler struct {
	repo         repository.ConfigurationRepository
	pollInterval time.Duration
	adsClient    *amznads.AmazonAdsAPIClient
}

// NewScheduler creates a new scheduler
func NewScheduler(cfg *Config) *Scheduler {
	return &Scheduler{
		repo:         cfg.ScheduleConfigRepo,
		pollInterval: cfg.PollInterval,
		adsClient:    cfg.AdsClient,
	}
}

// Start begins the scheduler in a goroutine
// It polls for due configurations at the specified interval
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	log.Printf("Scheduler started with poll interval: %v", s.pollInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler shutting down...")
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

// poll checks for due configurations and processes them
func (s *Scheduler) poll(ctx context.Context) {
	due, err := s.repo.GetDue()
	if err != nil {
		log.Printf("Error processing due configurations: %v", err)
		return
	}

	err = handleDueScheduleConfigurations(ctx, due, s.adsClient)
	if err != nil {
		log.Printf("Error processing due configurations: %v", err)
		return
	}
}
