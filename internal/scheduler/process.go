package scheduler

import (
	"context"

	amznads "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/domain"
)

func handleDueScheduleConfigurations(ctx context.Context, due []*domain.ScheduleConfiguration, adsClient *amznads.AmazonAdsAPIClient) error {
	return nil
}
