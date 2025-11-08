package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/LittleAksMax/bids-service/internal/handler"
)

// Middleware defines the type for HTTP Middleware functions
// Effectively, just a function that takes and returns an http.Handler
type Middleware func(next http.Handler) http.Handler

// ValidateBody is a middleware that validates the request body for ScheduleConfiguration
// It parses JSON, validates all required fields, and stores the validated config in context
func ValidateBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only validate POST requests with JSON content
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			http.Error(w, `{"error": "Content-Type must be application/json"}`, http.StatusBadRequest)
			return
		}

		// Parse JSON body
		var req handler.ScheduleConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			err := json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid JSON body: " + err.Error(),
			})
			if err != nil {
				return
			}
			return
		}

		// Validate required fields
		if validationErrors := req.Validate(); len(validationErrors) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  "Validation failed",
				"fields": validationErrors,
			})
			return
		}

		// Store validated config in context for handler to use
		ctx := context.WithValue(r.Context(), handler.ScheduleConfigKey, &req)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const ApiKeyHeader = "X-Api-Key"

func RequireAccessKey(apiKey string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(ApiKeyHeader)
			if key != apiKey {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
