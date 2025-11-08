package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/LittleAksMax/bids-service/internal/handler"
	"github.com/LittleAksMax/bids-service/internal/repository"
	"github.com/LittleAksMax/bids-service/internal/scheduler"
	"github.com/LittleAksMax/bids-service/internal/server"
	"github.com/LittleAksMax/bids-service/internal/service"
)

func main() {
	mode := os.Getenv("ENV")
	if mode == "development" {
		if err := godotenv.Load(".env.Dev"); err != nil {
			log.Panicf("Error loading .env.Dev file: %v", err)
		}
	} else if mode != "production" {
		log.Panic("ENV environment variable must be set to 'development' or 'production'")
	}
	log.Printf("Running in mode: '%s'", mode)

	// Load configuration
	cfg := LoadApiConfig()

	pollInterval := 15 * time.Minute // Default poll interval

	// Initialize dependencies (Dependency Injection)
	configRepo := repository.NewInMemoryConfigRepository()

	// Service layer
	configService := service.NewConfigurationService(configRepo)

	// Handler layer
	configHandler := handler.NewConfigHandler(configService)

	// Server
	httpServer := server.NewServer(cfg, configHandler)

	// Scheduler
	schedulerInstance := scheduler.NewScheduler(configService, pollInterval)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wait group for goroutines
	var wg sync.WaitGroup

	// Start HTTP server in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Start(ctx); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start scheduler in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		schedulerInstance.Start(ctx)
	}()

	// Wait for interrupt signal for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Received shutdown signal, initiating graceful shutdown...")

	// Cancel context to signal all goroutines to stop
	cancel()

	// Wait for all goroutines to finish
	wg.Wait()

	log.Println("Application shutdown complete")
}
