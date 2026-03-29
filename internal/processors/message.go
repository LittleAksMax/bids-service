package processors

import (
	adsapi "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/google/uuid"
)

type ProcessMessage struct {
	UserID  uuid.UUID
	Profile services.RegionProfile
	Report  *adsapi.Report
}
