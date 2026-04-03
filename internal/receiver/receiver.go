package receiver

import (
	"context"
	"errors"
	"io"
	"time"

	adsapi "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/processors"
	"github.com/LittleAksMax/bids-service/internal/profile_cache"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/LittleAksMax/bids-util/logging"
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

func NewReceiver(parent context.Context, userService *services.UserServiceClient, workers []*processors.Processor, profileCache profile_cache.ProfileCache, logger *logging.Logger) *Receiver {
	if logger == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(parent)
	receiver := &Receiver{
		userService:  userService,
		workers:      workers,
		pollInterval: time.Minute,
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
		r.logger.Errorf("Poll failed: %v", err)
	}
}

func (r *Receiver) pollOnce(ctx context.Context) error {
	schedules, err := r.userService.GetDueSchedules(ctx)
	if err != nil {
		return err
	}

	for _, schedule := range schedules {
		if err := r.handleSchedule(ctx, &schedule); err != nil {
			r.logger.Errorf("[UserID: %s; ProfileID: %d] Failed to handle schedule: %v", schedule.UserID, schedule.ProfileID, err)
		}
	}

	return nil
}

func NewReceiverLogger(writers ...io.Writer) *logging.Logger {
	return logging.NewLogger("[Receiver]", writers...)
}
