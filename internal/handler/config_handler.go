package handler

import (
	"encoding/json"
	"net/http"

	"github.com/LittleAksMax/bids-service/internal/repository"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	// ScheduleConfigKey is the context key for validated schedule configuration
	ScheduleConfigKey contextKey = "scheduleConfig"
)

// ConfigHandler handles HTTP requests for configurations
type ConfigHandler struct {
	repo repository.ConfigurationRepository
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(repo repository.ConfigurationRepository) *ConfigHandler {
	return &ConfigHandler{
		repo: repo,
	}
}

// HandleScheduleUpdate handles POST requests to update/create schedule configurations
func (h *ConfigHandler) HandleScheduleUpdate(w http.ResponseWriter, r *http.Request) {
	// Retrieve validated config from context (set by middleware)
	req, ok := r.Context().Value(ScheduleConfigKey).(*ScheduleConfigRequest)
	if !ok {
		http.Error(w, `{"error": "Invalid request context"}`, http.StatusInternalServerError)
		return
	}

	// Convert DTO to domain entity
	config := req.ToDomain()

	// TODO: Implement actual business logic
	// err := h.service.CreateConfiguration(config)
	// if err != nil {
	//     http.Error(w, err.Error(), http.StatusInternalServerError)
	//     return
	// }

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Schedule configuration created successfully",
		"data": map[string]interface{}{
			"userId":      config.UserID,
			"campaignId":  config.CampaignID,
			"marketplace": config.Marketplace,
			"interval":    req.Interval,
			"dueAt":       config.DueAt,
		},
	})
}
