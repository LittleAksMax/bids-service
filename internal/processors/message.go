package processors

import (
	"time"

	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/google/uuid"
)

type ProcessMessage struct {
	UserID          uuid.UUID
	Profile         services.RegionProfile
	RefreshToken    string
	DueAt           time.Time
	IntervalMinutes int64
}
