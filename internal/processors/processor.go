package processors

import (
	"context"

	adsapi "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/logging"
	"github.com/LittleAksMax/bids-service/internal/message_queue"
	"github.com/LittleAksMax/bids-service/internal/services"
)

type Processor struct {
	ID       int
	messages chan ProcessMessage

	messageQueue message_queue.MessageQueue

	adsClient     *adsapi.AmazonAdsAPIClient
	userService   *services.UserServiceClient
	policyService *services.PolicyServiceClient
	logger        *logging.Logger

	cancel context.CancelFunc
	done   chan struct{}
}

func NewProcessor(parent context.Context, procID, bufferSize int, adsClient *adsapi.AmazonAdsAPIClient, userService *services.UserServiceClient, policyService *services.PolicyServiceClient, mq message_queue.MessageQueue, logger *logging.Logger) *Processor {
	if logger == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(parent)

	processor := &Processor{
		ID:            procID,
		messages:      make(chan ProcessMessage, bufferSize),
		messageQueue:  mq,
		adsClient:     adsClient,
		userService:   userService,
		policyService: policyService,
		logger:        logger,
		cancel:        cancel,
		done:          make(chan struct{}),
	}

	go processor.run(ctx)

	return processor
}

func (p *Processor) Notify(msg ProcessMessage) {
	p.messages <- msg
}

func (p *Processor) run(ctx context.Context) {
	defer func() {
		if p.adsClient != nil {
			p.adsClient.CloseIdleConnections()
		}
		_ = p.logger.Close()
		close(p.done)
	}()

	for {
		select {
		// Stop consuming if we are done
		case <-ctx.Done():
			return
		case message, ok := <-p.messages:
			if !ok {
				return
			}

			p.handleMessage(ctx, message)
		}
	}
}
