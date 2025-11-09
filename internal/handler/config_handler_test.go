package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LittleAksMax/bids-service/internal/domain"
)

const testPollingInterval = 60 * time.Minute

// Mock repository for testing handlers
type mockRepo struct {
	getDueFunc                   func() ([]*domain.ScheduleConfiguration, error)
	getByUserIDFunc              func(userID string) ([]*domain.ScheduleConfiguration, bool)
	getByUserIDAndCampaignIDFunc func(userID, campaignID string) ([]*domain.ScheduleConfiguration, bool)
	putFunc                      func(config *domain.ScheduleConfiguration) error
}

func (m *mockRepo) GetDue() ([]*domain.ScheduleConfiguration, error) {
	if m.getDueFunc != nil {
		return m.getDueFunc()
	}
	return nil, nil
}

func (m *mockRepo) GetByUserID(userID string) ([]*domain.ScheduleConfiguration, bool) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(userID)
	}
	return nil, false
}

func (m *mockRepo) GetByUserIDAndCampaignID(userID, campaignID string) ([]*domain.ScheduleConfiguration, bool) {
	if m.getByUserIDAndCampaignIDFunc != nil {
		return m.getByUserIDAndCampaignIDFunc(userID, campaignID)
	}
	return nil, false
}

func (m *mockRepo) Put(config *domain.ScheduleConfiguration) error {
	if m.putFunc != nil {
		return m.putFunc(config)
	}
	return nil
}

func TestHandleScheduleUpdate(t *testing.T) {
	pollingIntervalMinutes := int(testPollingInterval.Minutes())

	tests := []struct {
		name           string
		setupContext   func(*http.Request) *http.Request
		putFunc        func(config *domain.ScheduleConfiguration) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful schedule update",
			setupContext: func(r *http.Request) *http.Request {
				req := &ScheduleConfigRequest{
					UserID:      "user123",
					CampaignID:  "campaign456",
					Marketplace: "US",
					Interval:    pollingIntervalMinutes,
				}
				ctx := context.WithValue(r.Context(), ScheduleConfigKey, req)
				return r.WithContext(ctx)
			},
			putFunc: func(config *domain.ScheduleConfiguration) error {
				return nil
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `"status":"success"`,
		},
		{
			name: "missing context value",
			setupContext: func(r *http.Request) *http.Request {
				return r // No context value set
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Invalid request context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepo{putFunc: tt.putFunc}
			handler := NewConfigHandler(mockRepo)

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req = tt.setupContext(req)
			w := httptest.NewRecorder()

			handler.HandleScheduleUpdate(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedBody)) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandleGetByUserID(t *testing.T) {
	tests := []struct {
		name            string
		userID          string
		getByUserIDFunc func(userID string) ([]*domain.ScheduleConfiguration, bool)
		expectedStatus  int
		expectedBody    string
	}{
		{
			name:   "successful retrieval",
			userID: "user123",
			getByUserIDFunc: func(userID string) ([]*domain.ScheduleConfiguration, bool) {
				return []*domain.ScheduleConfiguration{
					{
						UserID:      userID,
						CampaignID:  "campaign1",
						Marketplace: "US",
						Interval:    testPollingInterval,
					},
				}, true
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"status":"success"`,
		},
		{
			name:   "user not found",
			userID: "nonexistent",
			getByUserIDFunc: func(userID string) ([]*domain.ScheduleConfiguration, bool) {
				return nil, false
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "No configurations found for user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepo{getByUserIDFunc: tt.getByUserIDFunc}
			handler := NewConfigHandler(mockRepo)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.userID, nil)
			req.SetPathValue("userId", tt.userID)
			w := httptest.NewRecorder()

			handler.HandleGetByUserID(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedBody)) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestHandleGetByUserIDAndCampaignID(t *testing.T) {
	tests := []struct {
		name                         string
		userID                       string
		campaignID                   string
		getByUserIDAndCampaignIDFunc func(userID, campaignID string) ([]*domain.ScheduleConfiguration, bool)
		expectedStatus               int
		expectedBody                 string
	}{
		{
			name:       "successful retrieval",
			userID:     "user123",
			campaignID: "campaign456",
			getByUserIDAndCampaignIDFunc: func(userID, campaignID string) ([]*domain.ScheduleConfiguration, bool) {
				return []*domain.ScheduleConfiguration{
					{
						UserID:      userID,
						CampaignID:  campaignID,
						Marketplace: "US",
						Interval:    testPollingInterval,
					},
				}, true
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `"status":"success"`,
		},
		{
			name:       "not found",
			userID:     "user123",
			campaignID: "nonexistent",
			getByUserIDAndCampaignIDFunc: func(userID, campaignID string) ([]*domain.ScheduleConfiguration, bool) {
				return nil, false
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "No configurations found for user and campaign",
		},
		{
			name:           "empty campaign ID",
			userID:         "user123",
			campaignID:     "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Campaign ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockRepo{getByUserIDAndCampaignIDFunc: tt.getByUserIDAndCampaignIDFunc}
			handler := NewConfigHandler(mockRepo)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.userID+"/"+tt.campaignID, nil)
			req.SetPathValue("userId", tt.userID)
			req.SetPathValue("campaignId", tt.campaignID)
			w := httptest.NewRecorder()

			handler.HandleGetByUserIDAndCampaignID(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedBody)) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestScheduleConfigRequest_Validate(t *testing.T) {
	pollingIntervalMinutes := int(testPollingInterval.Minutes())
	invalidInterval := pollingIntervalMinutes + 1 // Not a multiple

	tests := []struct {
		name          string
		request       ScheduleConfigRequest
		expectErrors  bool
		errorContains []string
	}{
		{
			name: "valid request",
			request: ScheduleConfigRequest{
				UserID:      "user123",
				CampaignID:  "campaign456",
				Marketplace: "US",
				Interval:    pollingIntervalMinutes,
			},
			expectErrors: false,
		},
		{
			name: "missing user ID",
			request: ScheduleConfigRequest{
				CampaignID:  "campaign456",
				Marketplace: "US",
				Interval:    pollingIntervalMinutes,
			},
			expectErrors:  true,
			errorContains: []string{"userId"},
		},
		{
			name: fmt.Sprintf("invalid interval - not multiple of %d", pollingIntervalMinutes),
			request: ScheduleConfigRequest{
				UserID:      "user123",
				CampaignID:  "campaign456",
				Marketplace: "US",
				Interval:    invalidInterval,
			},
			expectErrors:  true,
			errorContains: []string{"interval"},
		},
		{
			name: "zero interval",
			request: ScheduleConfigRequest{
				UserID:      "user123",
				CampaignID:  "campaign456",
				Marketplace: "US",
				Interval:    0,
			},
			expectErrors:  true,
			errorContains: []string{"interval"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.request.Validate(testPollingInterval)

			if tt.expectErrors && len(errors) == 0 {
				t.Error("expected validation errors, got none")
			}

			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("expected no validation errors, got: %v", errors)
			}

			for _, errorKey := range tt.errorContains {
				if _, ok := errors[errorKey]; !ok {
					t.Errorf("expected error for field %q, but not found", errorKey)
				}
			}
		})
	}
}

func TestScheduleConfigRequest_ToDomain(t *testing.T) {
	pollingIntervalMinutes := int(testPollingInterval.Minutes())
	// Use a valid multiple of polling interval for testing
	testInterval := pollingIntervalMinutes * 2 // e.g., 120 if polling is 60

	req := ScheduleConfigRequest{
		UserID:      "user123",
		CampaignID:  "campaign456",
		Marketplace: "UK",
		Interval:    testInterval,
	}

	config := req.ToDomain()

	if config.UserID != req.UserID {
		t.Errorf("expected UserID %q, got %q", req.UserID, config.UserID)
	}

	if config.CampaignID != req.CampaignID {
		t.Errorf("expected CampaignID %q, got %q", req.CampaignID, config.CampaignID)
	}

	if config.Marketplace != req.Marketplace {
		t.Errorf("expected Marketplace %q, got %q", req.Marketplace, config.Marketplace)
	}

	expectedInterval := time.Duration(testInterval) * time.Minute
	if config.Interval != expectedInterval {
		t.Errorf("expected Interval %v, got %v", expectedInterval, config.Interval)
	}

	// Verify DueAt is in the future
	now := time.Now().UTC()
	if config.DueAt.Before(now) {
		t.Error("DueAt should be in the future")
	}

	if int(config.Interval.Minutes())%pollingIntervalMinutes != 0 {
		t.Errorf("DueAt minutes should be multiple of %d, got %d", pollingIntervalMinutes, config.DueAt.Minute())
	}
}

// TODO: test marketplaces being valid (against SDK?)
