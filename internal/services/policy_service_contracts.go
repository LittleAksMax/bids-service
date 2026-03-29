package services

type Policy struct {
	ID          string `json:"id,omitempty"`
	UserID      string `json:"user_id"`
	Marketplace string `json:"marketplace"`
	Name        string `json:"name"`
	Script      string `json:"script"`
}
