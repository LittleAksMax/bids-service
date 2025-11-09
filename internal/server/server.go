package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/LittleAksMax/bids-service/internal/handler"
)

// Server wraps the HTTP server
type Server struct {
	httpServer   *http.Server
	handler      *handler.ConfigHandler
	pollInterval time.Duration
}

type Config struct {
	ApiKey       string
	Port         int
	PollInterval time.Duration
}

// NewServer creates a new HTTP server
func NewServer(cfg *Config, handler *handler.ConfigHandler) *Server {
	srv := &Server{
		handler:      handler,
		pollInterval: cfg.PollInterval,
	}

	mux := srv.routes(cfg.ApiKey)

	srv.httpServer = &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%s", strconv.Itoa(cfg.Port)),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv
}

// routes sets up the HTTP routes
func (s *Server) routes(apiKey string) http.Handler {
	mux := chi.NewRouter()

	// Middleware
	mux.Use(middleware.Heartbeat("/ping"))
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	// Routes - POST endpoint with validation middleware
	requireAccessKeyFunc := RequireAccessKey(apiKey)
	mux.With(ValidateBody(s.pollInterval)).With(requireAccessKeyFunc).Post("/", s.handler.HandleScheduleUpdate)

	// GET endpoints with access key middleware
	mux.With(requireAccessKeyFunc).Get("/{userId}", s.handler.HandleGetByUserID)
	mux.With(requireAccessKeyFunc).Get("/{userId}/{campaignId}", s.handler.HandleGetByUserIDAndCampaignID)

	return mux
}

// Start starts the HTTP server in a goroutine
func (s *Server) Start(ctx context.Context) error {
	errChan := make(chan error, 1)

	go func() {
		log.Printf("HTTP server starting on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return s.Shutdown()
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	log.Println("HTTP server shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}
