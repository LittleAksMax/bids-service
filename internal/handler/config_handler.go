package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/LittleAksMax/bids-service/internal/service"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	// ScheduleConfigKey is the context key for validated schedule configuration
	ScheduleConfigKey contextKey = "scheduleConfig"
)

// ConfigHandler handles HTTP requests for configurations
type ConfigHandler struct {
	service *service.ConfigurationService
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(service *service.ConfigurationService) *ConfigHandler {
	return &ConfigHandler{
		service: service,
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

	// TODO: Call service layer to process the configuration
	// For now, just log it
	log.Printf("Received valid schedule config: UserID=%s, CampaignID=%s, Marketplace=%s, Interval=%d mins, DueAt=%s",
		config.UserID, config.CampaignID, config.Marketplace, req.Interval, config.DueAt)

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
