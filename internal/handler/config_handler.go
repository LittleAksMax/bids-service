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
	schedConfig := req.ToDomain()

	err := h.repo.Put(schedConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Schedule configuration created successfully",
		"data": map[string]interface{}{
			"userId":      schedConfig.UserID,
			"campaignId":  schedConfig.CampaignID,
			"marketplace": schedConfig.Marketplace,
			"interval":    schedConfig.Interval,
			"dueAt":       schedConfig.DueAt,
		},
	})
}

// HandleGetByUserID handles GET requests to retrieve all configurations for a user
func (h *ConfigHandler) HandleGetByUserID(w http.ResponseWriter, r *http.Request) {
	// Extract userID from URL path parameter
	userID := r.PathValue("userId")
	if userID == "" {
		http.Error(w, `{"error": "User ID is required"}`, http.StatusBadRequest)
		return
	}

	configs, found := h.repo.GetByUserID(userID)
	if !found {
		http.Error(w, `{"error": "No configurations found for user"}`, http.StatusNotFound)
		return
	}

	// Return configurations
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   configs,
	})
}

// HandleGetByUserIDAndCampaignID handles GET requests to retrieve configurations for a user and campaign
func (h *ConfigHandler) HandleGetByUserIDAndCampaignID(w http.ResponseWriter, r *http.Request) {
	// Extract userID and campaignID from URL path parameters
	userID := r.PathValue("userId")
	campaignID := r.PathValue("campaignId")

	if userID == "" {
		http.Error(w, `{"error": "User ID is required"}`, http.StatusBadRequest)
		return
	}

	if campaignID == "" {
		http.Error(w, `{"error": "Campaign ID is required"}`, http.StatusBadRequest)
		return
	}

	configs, found := h.repo.GetByUserIDAndCampaignID(userID, campaignID)
	if !found {
		http.Error(w, `{"error": "No configurations found for user and campaign"}`, http.StatusNotFound)
		return
	}

	// Return configurations
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   configs,
	})
}
