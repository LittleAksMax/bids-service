package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LittleAksMax/bids-service/internal/handler"
)

func TestValidateBody(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		pollInterval   time.Duration
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid request",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"userId": "user123", "campaignId": "campaign456", "marketplace": "US", "interval": 60}`,
			pollInterval:   60 * time.Minute,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-POST request passes through",
			method:         http.MethodGet,
			contentType:    "application/json",
			body:           "",
			pollInterval:   60 * time.Minute,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong content type",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           `{"userId": "user123"}`,
			pollInterval:   60 * time.Minute,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Content-Type must be application/json",
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"userId": "user123"`,
			pollInterval:   60 * time.Minute,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON body",
		},
		{
			name:           "missing required fields",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"userId": "user123", "interval": 30}`,
			pollInterval:   60 * time.Minute,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Validation failed",
		},
		{
			name:           "invalid interval - not multiple of polling interval",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"userId": "user123", "campaignId": "campaign456", "marketplace": "US", "interval": 30}`,
			pollInterval:   60 * time.Minute,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   fmt.Sprintf("interval must be a multiple of polling interval (%d) minutes", 60),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that checks if context was set properly
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.method == http.MethodPost && tt.expectedStatus == http.StatusOK {
					// Verify context was set
					if _, ok := r.Context().Value(handler.ScheduleConfigKey).(*handler.ScheduleConfigRequest); !ok {
						t.Error("expected ScheduleConfigRequest in context")
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			middleware := ValidateBody(tt.pollInterval)(nextHandler)

			req := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedBody)) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestRequireAccessKey(t *testing.T) {
	const testAPIKey = "test-api-key-12345"

	tests := []struct {
		name           string
		apiKey         string
		providedKey    string
		expectedStatus int
	}{
		{
			name:           "valid API key",
			apiKey:         testAPIKey,
			providedKey:    testAPIKey,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid API key",
			apiKey:         testAPIKey,
			providedKey:    "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing API key",
			apiKey:         testAPIKey,
			providedKey:    "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequireAccessKey(tt.apiKey)(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.providedKey != "" {
				req.Header.Set(ApiKeyHeader, tt.providedKey)
			}
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK && w.Code != http.StatusOK {
				t.Error("expected next handler to be called")
			}
		})
	}
}

func TestValidateBodyWithRequireAccessKey(t *testing.T) {
	const testAPIKey = "test-api-key"
	const testPollInterval = 60 * time.Minute

	// Test chaining both middlewares together
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that both middlewares ran successfully
		if _, ok := r.Context().Value(handler.ScheduleConfigKey).(*handler.ScheduleConfigRequest); !ok {
			t.Error("expected ScheduleConfigRequest in context")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})

	// Chain middlewares
	middlewareChain := ValidateBody(testPollInterval)(RequireAccessKey(testAPIKey)(nextHandler))

	validBody := `{
		"userId": "user123",
		"campaignId": "campaign456",
		"marketplace": "US",
		"interval": 60
	}`

	tests := []struct {
		name           string
		apiKey         string
		body           string
		expectedStatus int
	}{
		{
			name:           "both middlewares pass",
			apiKey:         testAPIKey,
			body:           validBody,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "auth fails",
			apiKey:         "wrong-key",
			body:           validBody,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(ApiKeyHeader, tt.apiKey)
			w := httptest.NewRecorder()

			middlewareChain.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
