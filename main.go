package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	adsapi "github.com/LittleAksMax/amazon-ads-api-sdk-go"
	"github.com/LittleAksMax/bids-service/internal/config"
	"github.com/LittleAksMax/bids-service/internal/processors"
	"github.com/LittleAksMax/bids-service/internal/profile_cache"
	"github.com/LittleAksMax/bids-service/internal/receiver"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/joho/godotenv"
)

const (
	ModeDevelopment = "development"
	ModeProduction  = "production"
)

func main() {
	mode := os.Getenv("MODE")
	if mode == ModeDevelopment {
		if err := godotenv.Load(".env.Dev"); err != nil {
			log.Fatalf("error loading .env file: %v\n", err)
		}
	} else if mode == ModeProduction {
	} else {
		log.Fatalf("invalid mode %s\n", mode)
	}

	cfg := config.Load()

	// Make sure logging directory exists
	if err := os.MkdirAll(cfg.WorkersConfig.LogPath, 0o755); err != nil {
		log.Fatalf("error creating log directory: %v\n", err)
	}

	// Create context with SIGTERM signal available for graceful shutdown in case of interrupt
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	if err := run(ctx, cancel, cfg); err != nil {
		log.Fatalf("%v", err)
	}
}

func run(ctx context.Context, cancel context.CancelFunc, cfg *config.Config) error {
	workers := make([]*processors.Processor, 0, cfg.WorkersConfig.NumProcessors)
	var pollingReceiver *receiver.Receiver

	// Make internal service clients
	userServiceClient, err := newUserServiceClient(cfg)
	if err != nil {
		return err
	}
	policyServiceClient, err := newPolicyServiceClient(cfg)
	if err != nil {
		return err
	}

	profileCache, err := profile_cache.NewProfileCache(ctx, userServiceClient)
	if err != nil {
		return fmt.Errorf("error creating profile cache: %v", err)
	}

	// Graceful shutdown function
	defer func() {
		log.Print("Shutting down, cancelling context")
		cancel()
		<-ctx.Done()

		if pollingReceiver != nil {
			log.Print("Shutting down polling receiver")
			pollingReceiver.Interrupt()
			pollingReceiver.Wait()
		}
		log.Print("Shutting down processors")
		processors.InterruptAll(workers)
		processors.WaitAll(workers)
		log.Print("Shutting down profile cache")
		_ = profileCache.Close()
		log.Print("Closing user service")
		userServiceClient.CloseIdleConnections()
		log.Print("Closing policy service")
		policyServiceClient.CloseIdleConnections()
		log.Print("Shutdown finished")
	}()

	for i := 0; i < cfg.WorkersConfig.NumProcessors; i++ {
		processorAdsClient, err := newAmazonAdsClient(cfg)
		if err != nil {
			return fmt.Errorf("could not create processor ads client %d: %v", i, err)
		}

		processorLogFile, err := openLogFile(cfg.WorkersConfig.LogPath, fmt.Sprintf("processor-%d.log", i))
		if err != nil {
			return err
		}

		proc := processors.NewProcessor(
			ctx,
			i,
			cfg.WorkersConfig.BufferSize,
			processorAdsClient,
			userServiceClient,
			policyServiceClient,
			processors.NewProcessorLogger(i, os.Stdout, processorLogFile),
		)
		if proc == nil {
			_ = processorLogFile.Close()
			return fmt.Errorf("could not create all workers: created %d of %d", len(workers), cfg.WorkersConfig.NumProcessors)
		}
		workers = append(workers, proc)
	}

	receiverAdsClient, err := newAmazonAdsClient(cfg)
	if err != nil {
		return err
	}

	receiverLogFile, err := openLogFile(cfg.WorkersConfig.LogPath, "receiver.log")
	if err != nil {
		return err
	}

	pollingReceiver = receiver.NewReceiver(
		ctx,
		userServiceClient,
		receiverAdsClient,
		workers,
		profileCache,
		receiver.NewReceiverLogger(os.Stdout, receiverLogFile),
	)
	if pollingReceiver == nil {
		_ = receiverLogFile.Close()
		return fmt.Errorf("could not create receiver: %v", err)
	}

	<-ctx.Done()

	return nil
}

func newAmazonAdsClient(cfg *config.Config) (*adsapi.AmazonAdsAPIClient, error) {
	authConfig := adsapi.NewAmazonAuthAPIConfig(cfg.AmazonAdsConfig.ClientID, cfg.AmazonAdsConfig.ClientSecret, "")
	authClient, err := adsapi.NewAmazonAuthClient(authConfig, adsapi.AmazonRegions.Europe)
	if err != nil {
		return nil, fmt.Errorf("error initialising auth client: %v", err)
	}

	client, err := adsapi.NewAmazonAdsAPIClient(&adsapi.Configuration{
		AuthClient: authClient,
		Region:     adsapi.AmazonRegions.Europe,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func newUserServiceClient(cfg *config.Config) (*services.UserServiceClient, error) {
	client, err := services.NewUserServiceClient(
		&http.Client{Timeout: 15 * time.Second},
		cfg.UserServiceConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create user service client: %v", err)
	}

	return client, nil
}

func newPolicyServiceClient(cfg *config.Config) (*services.PolicyServiceClient, error) {
	client, err := services.NewPolicyServiceClient(
		&http.Client{Timeout: 15 * time.Second},
		cfg.PolicyServiceConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create user service client: %v", err)
	}

	return client, nil
}

func openLogFile(dir, name string) (*os.File, error) {
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("could not open log file %s: %v", name, err)
	}

	return file, nil
}
