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
	"github.com/LittleAksMax/bids-service/internal/repository"
)

const testAPIKey = "test-api-key-123"

func setupTestServer(t *testing.T) (*Server, repository.ConfigurationRepository) {
	repo := repository.NewInMemoryConfigRepository()
	h := handler.NewConfigHandler(repo)

	cfg := &Config{
		ApiKey:       testAPIKey,
		Port:         8080,
		PollInterval: 60 * time.Minute,
	}

	server := NewServer(cfg, h)
	return server, repo
}

func TestServerIntegration_PostAndGetFlow(t *testing.T) {
	server, _ := setupTestServer(t)

	// Test 1: Create a schedule configuration
	t.Run("POST schedule configuration", func(t *testing.T) {
		body := `{
			"userId": "testuser",
			"campaignId": "testcampaign",
			"marketplace": "US",
			"interval": 60
		}`

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(ApiKeyHeader, testAPIKey)
		w := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["status"] != "success" {
			t.Errorf("expected status success, got %v", response["status"])
		}
	})

	// Test 2: Retrieve the configuration by user ID
	t.Run("GET by user ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/testuser", nil)
		req.Header.Set(ApiKeyHeader, testAPIKey)
		req.SetPathValue("userId", "testuser")
		w := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["status"] != "success" {
			t.Errorf("expected status success, got %v", response["status"])
		}

		data, ok := response["data"].([]interface{})
		if !ok || len(data) == 0 {
			t.Error("expected data array with at least one item")
		}
	})

	// Test 3: Retrieve by user ID and campaign ID
	t.Run("GET by user ID and campaign ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/testuser/testcampaign", nil)
		req.Header.Set(ApiKeyHeader, testAPIKey)
		req.SetPathValue("userId", "testuser")
		req.SetPathValue("campaignId", "testcampaign")
		w := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response["status"] != "success" {
			t.Errorf("expected status success, got %v", response["status"])
		}
	})
}

func TestServerIntegration_MultipleConfigurations(t *testing.T) {
	server, _ := setupTestServer(t)

	// Create multiple configurations for the same user
	configs := []struct {
		campaignID  string
		marketplace string
		interval    int
	}{
		{"campaign1", "US", 60},
		{"campaign1", "UK", 120},
		{"campaign2", "US", 180},
	}

	for _, cfg := range configs {
		body := map[string]interface{}{
			"userId":      "multiuser",
			"campaignId":  cfg.campaignID,
			"marketplace": cfg.marketplace,
			"interval":    cfg.interval,
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(ApiKeyHeader, testAPIKey)
		w := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("failed to create config: %s", w.Body.String())
		}
	}

	// Test: Get all configs for user
	t.Run("GET all configs for user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/multiuser", nil)
		req.Header.Set(ApiKeyHeader, testAPIKey)
		req.SetPathValue("userId", "multiuser")
		w := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		json.NewDecoder(w.Body).Decode(&response)

		data := response["data"].([]interface{})
		if len(data) != 3 {
			t.Errorf("expected 3 configurations, got %d", len(data))
		}
	})

	// Test: Get configs for specific campaign
	t.Run("GET configs for campaign1", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/multiuser/campaign1", nil)
		req.Header.Set(ApiKeyHeader, testAPIKey)
		req.SetPathValue("userId", "multiuser")
		req.SetPathValue("campaignId", "campaign1")
		w := httptest.NewRecorder()

		server.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		json.NewDecoder(w.Body).Decode(&response)

		data := response["data"].([]interface{})
		if len(data) != 2 {
			t.Errorf("expected 2 configurations for campaign1, got %d", len(data))
		}
	})
}

func TestServerIntegration_Authentication(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "valid API key",
			apiKey:         testAPIKey,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid API key",
			apiKey:         "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing API key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := `{
				"userId": "user",
				"campaignId": "campaign",
				"marketplace": "US",
				"interval": 60
			}`

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.apiKey != "" {
				req.Header.Set(ApiKeyHeader, tt.apiKey)
			}
			w := httptest.NewRecorder()

			server.httpServer.Handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestServerIntegration_ValidationErrors(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "missing userId",
			body:           `{"campaignId":"c1","marketplace":"US","interval":60}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "Validation failed",
		},
		{
			name:           "invalid interval - not multiple of polling interval",
			body:           `{"userId":"u1","campaignId":"c1","marketplace":"US","interval":30}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  fmt.Sprintf("interval must be a multiple of polling interval (%d) minutes", int(server.pollInterval.Minutes())),
		},
		{
			name:           "zero interval",
			body:           `{"userId":"u1","campaignId":"c1","marketplace":"US","interval":0}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "Validation failed",
		},
		{
			name:           "invalid JSON",
			body:           `{"userId":"u1"`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "Invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(ApiKeyHeader, testAPIKey)
			w := httptest.NewRecorder()

			server.httpServer.Handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if !bytes.Contains(w.Body.Bytes(), []byte(tt.errorContains)) {
				t.Errorf("expected error to contain %q, got %q", tt.errorContains, w.Body.String())
			}
		})
	}
}

func TestServerIntegration_NotFoundErrors(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name string
		url  string
		path map[string]string
	}{
		{
			name: "non-existent user",
			url:  "/nonexistentuser",
			path: map[string]string{"userId": "nonexistentuser"},
		},
		{
			name: "non-existent user and campaign",
			url:  "/nonexistentuser/nonexistentcampaign",
			path: map[string]string{
				"userId":     "nonexistentuser",
				"campaignId": "nonexistentcampaign",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req.Header.Set(ApiKeyHeader, testAPIKey)
			for key, val := range tt.path {
				req.SetPathValue(key, val)
			}
			w := httptest.NewRecorder()

			server.httpServer.Handler.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
			}

			if !bytes.Contains(w.Body.Bytes(), []byte("No configurations found")) {
				t.Errorf("expected 'No configurations found' in response, got %q", w.Body.String())
			}
		})
	}
}

func TestServerIntegration_UpdateExistingConfig(t *testing.T) {
	server, repo := setupTestServer(t)

	// Create initial config
	initialBody := `{
		"userId": "updateuser",
		"campaignId": "updatecampaign",
		"marketplace": "US",
		"interval": 60
	}`

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(initialBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ApiKeyHeader, testAPIKey)
	w := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create initial config: %s", w.Body.String())
	}

	// Update the same config (same user, campaign, marketplace)
	updatedBody := `{
		"userId": "updateuser",
		"campaignId": "updatecampaign",
		"marketplace": "US",
		"interval": 120
	}`

	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(updatedBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(ApiKeyHeader, testAPIKey)
	w = httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("failed to update config: %s", w.Body.String())
	}

	// Verify only one config exists with updated interval
	configs, _ := repo.GetByUserID("updateuser")
	if len(configs) != 1 {
		t.Errorf("expected 1 config after update, got %d", len(configs))
	}

	if configs[0].Interval != 120*time.Minute {
		t.Errorf("expected interval 120m, got %v", configs[0].Interval)
	}
}

func TestServerIntegration_Heartbeat(t *testing.T) {
	server, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for /ping, got %d", http.StatusOK, w.Code)
	}
}
