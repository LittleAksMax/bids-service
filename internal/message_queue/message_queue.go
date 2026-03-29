package message_queue

import (
	eval "github.com/LittleAksMax/bidscript/evaluator"
	"github.com/google/uuid"
)

type ResultMessage struct {
	UserID     uuid.UUID   `json:"user_id"`
	ProfileID  int64       `json:"profile_id"`
	CampaignID string      `json:"campaign_id"`
	AdGroupID  string      `json:"adgroup_id"`
	Result     eval.Result `json:"result"`
}

type MessageQueue interface {
	Publish(result *eval.Result) error
	Close() error
}
