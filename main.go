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

	amznads "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/handler"
	"github.com/LittleAksMax/bids-service/internal/repository"
	"github.com/LittleAksMax/bids-service/internal/scheduler"
	"github.com/LittleAksMax/bids-service/internal/server"
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
	serverCfg := loadServerConfig()

	// SDK authentication setup
	refreshToken := RefreshToken{}
	adsAuthCfg := amznads.NewAmazonAuthAPIConfig(
		os.Getenv("AMZNADS_CLIENT_ID"),
		os.Getenv("AMZNADS_CLIENT_SECRET"),
		"", // no redirect URI
	)
	amznAuthClient, err := amznads.NewAmazonAuthClient(adsAuthCfg, amznads.AmazonRegions.Europe)
	if err != nil {
		log.Panicf("Error initializing Amazon Auth API client: %v", err)
	}
	amznAdsClient, err := amznads.NewAmazonAdsAPIClient(amznAuthClient, amznads.AmazonRegions.Europe)
	if err != nil {
		log.Panicf("Error initializing Amazon Ads API client: %v", err)
	}
	// TODO: maybe we should implement a provider that refetches when expired
	amznAdsClient.SetRefreshToken(refreshToken.Get())
	pollInterval := 1 * time.Hour

	// Initialize dependencies (Dependency Injection)
	configRepo := repository.NewInMemoryConfigRepository()

	// Handler layer
	configHandler := handler.NewConfigHandler(configRepo)

	// Server
	httpServer := server.NewServer(serverCfg, configHandler)

	// Scheduler
	schedulerInstance := scheduler.NewScheduler(&scheduler.Config{
		ScheduleConfigRepo: configRepo,
		PollInterval:       pollInterval,
		AdsClient:          amznAdsClient,
	})

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
