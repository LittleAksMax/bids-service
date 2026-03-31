package processors

import (
	"context"
	"fmt"
	"io"

	adsapi "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/LittleAksMax/bids-util/logging"
)

type Processor struct {
	ID       int
	messages chan ProcessMessage

	adsClient     *adsapi.AmazonAdsAPIClient
	userService   *services.UserServiceClient
	policyService *services.PolicyServiceClient
	logger        *logging.Logger

	cancel context.CancelFunc
	done   chan struct{}
}

func NewProcessor(parent context.Context, procID, bufferSize int, adsClient *adsapi.AmazonAdsAPIClient, userService *services.UserServiceClient, policyService *services.PolicyServiceClient, logger *logging.Logger) *Processor {
	if logger == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(parent)

	processor := &Processor{
		ID:            procID,
		messages:      make(chan ProcessMessage, bufferSize),
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

func NewProcessorLogger(id int, writers ...io.Writer) *logging.Logger {
	return logging.NewLogger(fmt.Sprintf("[Processor %d]", id), writers...)
}
