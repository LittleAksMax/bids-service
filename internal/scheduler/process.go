package scheduler

import (
	"context"
	"log"
	"time"

	amznads "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/domain"
)

func handleDueScheduleConfigurations(ctx context.Context, due []*domain.ScheduleConfiguration, adsClient *amznads.AmazonAdsAPIClient) error {
	for _, sched := range due {
		log.Println(sched)

		// TODO: update database
		sched.DueAt = time.Now().UTC().Add(sched.Interval)
	}
	return nil
}
