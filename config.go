package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/LittleAksMax/bids-service/internal/server"
)

func LoadApiConfig() *server.Config {
	return &server.Config{
		ApiKey: getStrFromEnv("API_KEY"),
		Port:   readPort("API_PORT"),
	}
}

func getStrFromEnv(key string) string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		log.Panicf("%s is not set in environment", key)
	}
	return valueStr
}

func getStrListFromEnv(key string) []string {
	valueStr := getStrFromEnv(key)
	values := strings.Split(valueStr, ",")
	return values
}

func getIntFromEnv(key string) int {
	valueStr := getStrFromEnv(key)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Panicf("%s couldn't be converted to int", key)
	}
	return value
}

func readPort(key string) int {
	port := getIntFromEnv(key)

	if port < 1024 || port > 65353 {
		log.Panicf("Error converting environment variable <%s> to int between 1024 and 65353", key)
	}
	return port
}
