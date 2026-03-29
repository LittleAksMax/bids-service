package config

import (
	mq "github.com/LittleAksMax/bids-service/internal/message_queue"
	"github.com/LittleAksMax/bids-service/internal/services"
	"github.com/LittleAksMax/bids-util/env"
)

type AmazonAdsConfig struct {
	ClientID     string
	ClientSecret string
}

type WorkersConfig struct {
	NumProcessors int
	BufferSize    int
	LogPath       string
}

type Config struct {
	MessageQueueConfig  *mq.Config
	AmazonAdsConfig     *AmazonAdsConfig
	WorkersConfig       *WorkersConfig
	UserServiceConfig   *services.ServiceConfig
	PolicyServiceConfig *services.ServiceConfig
}

// Load reads environment variables and returns a Config.
// Required PORT, ACCESS_TOKEN_SECRET, REFRESH_TOKEN_SECRET, VALIDATION_API_KEY, REDIS_HOST, REDIS_PORT, REDIS_PASSWORD
func Load() *Config {
	return &Config{
		MessageQueueConfig: &mq.Config{
			Host:     env.GetStrFromEnv("RABBITMQ_HOST"),
			Port:     env.ReadPort("RABBITMQ_PORT"),
			User:     env.GetStrFromEnv("RABBITMQ_USER"),
			Password: env.GetStrFromEnv("RABBITMQ_PASSWORD"),
			Queue:    env.GetStrFromEnv("RABBITMQ_QUEUE"),
		},
		AmazonAdsConfig: &AmazonAdsConfig{
			ClientID:     env.GetStrFromEnv("AMAZON_ADS_CLIENT_ID"),
			ClientSecret: env.GetStrFromEnv("AMAZON_ADS_CLIENT_SECRET"),
		},
		WorkersConfig: &WorkersConfig{
			NumProcessors: env.GetIntFromEnv("NUM_PROCESSORS"),
			BufferSize:    env.GetIntFromEnv("WORKER_BUFSIZE"),
			LogPath:       env.GetStrFromEnv("LOG_PATH"),
		},
		UserServiceConfig: &services.ServiceConfig{
			BaseURL:      env.GetStrFromEnv("USER_SERVICE_BASE_URL"),
			APIKey:       env.GetStrFromEnv("USER_SERVICE_API_KEY"),
			APIKeyHeader: env.GetStrFromEnv("USER_SERVICE_API_KEY_HEADER"),
		},
		PolicyServiceConfig: &services.ServiceConfig{
			BaseURL:      env.GetStrFromEnv("POLICY_SERVICE_BASE_URL"),
			APIKey:       env.GetStrFromEnv("POLICY_SERVICE_API_KEY"),
			APIKeyHeader: env.GetStrFromEnv("POLICY_SERVICE_API_KEY_HEADER"),
		},
	}
}
