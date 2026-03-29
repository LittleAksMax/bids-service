package receiver

import (
	"context"
	"errors"
	"time"

	adsapi "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/logging"
	"github.com/LittleAksMax/bids-service/internal/processors"
	"github.com/LittleAksMax/bids-service/internal/profile_cache"
	"github.com/LittleAksMax/bids-service/internal/services"
)

type Receiver struct {
	userService  *services.UserServiceClient
	adsClient    *adsapi.AmazonAdsAPIClient
	workers      []*processors.Processor
	pollInterval time.Duration
	profileCache profile_cache.ProfileCache
	logger       *logging.Logger

	cancel context.CancelFunc
	done   chan struct{}
}

func NewReceiver(parent context.Context, userService *services.UserServiceClient, adsClient *adsapi.AmazonAdsAPIClient, workers []*processors.Processor, profileCache profile_cache.ProfileCache, logger *logging.Logger) *Receiver {
	if logger == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(parent)
	receiver := &Receiver{
		userService:  userService,
		adsClient:    adsClient,
		workers:      workers,
		pollInterval: time.Second * 5, // TODO: Change back to time.Minute
		profileCache: profileCache,
		logger:       logger,
		cancel:       cancel,
		done:         make(chan struct{}),
	}

	go receiver.run(ctx)

	return receiver
}

func (r *Receiver) run(ctx context.Context) {
	defer func() {
		_ = r.logger.Close()
		close(r.done)
	}()

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.pollAndLog(ctx)
		}
	}
}

func (r *Receiver) pollAndLog(ctx context.Context) {
	if err := r.pollOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
		r.logger.Errorf("poll failed: %v", err)
	}
}

func (r *Receiver) pollOnce(ctx context.Context) error {
	schedules, err := r.userService.GetDueSchedules(ctx)
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		if err := r.handleSchedule(ctx, &schedule); err != nil {
			r.logger.Errorf("failed to handle profile %d: %v", schedule.ProfileID, err)
		}
	}

	return nil
}
