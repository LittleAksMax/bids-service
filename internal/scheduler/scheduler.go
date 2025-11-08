package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/LittleAksMax/bids-service/internal/service"
)

// Scheduler handles periodic polling and processing of due configurations
type Scheduler struct {
	service      *service.ConfigurationService
	pollInterval time.Duration
}

// NewScheduler creates a new scheduler
func NewScheduler(service *service.ConfigurationService, pollInterval time.Duration) *Scheduler {
	return &Scheduler{
		service:      service,
		pollInterval: pollInterval,
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
			s.poll()
		}
	}
}

// poll checks for due configurations and processes them
func (s *Scheduler) poll() {
	log.Println("Polling for due configurations...")

	// TODO: Implement polling logic
	err := s.service.ProcessDueConfigurations()
	if err != nil {
		log.Printf("Error processing due configurations: %v", err)
		return
	}

	log.Println("Poll completed successfully")
}
