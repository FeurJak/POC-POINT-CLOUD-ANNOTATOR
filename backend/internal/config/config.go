// Package config provides configuration management for the application.
package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application.
type Config struct {
	// Role specifies the service role: "gateway" or "handler"
	Role string

	// Server configuration
	ServerPort string

	// Handler service URL (used by gateway to forward requests)
	HandlerURL string

	// Database configuration
	DatabaseURL string

	// Redis configuration
	RedisURL string

	// Environment
	Environment string
}

// New creates a new Config with values from environment variables or defaults.
func New() *Config {
	return &Config{
		Role:        getEnv("SERVICE_ROLE", "gateway"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		HandlerURL:  getEnv("HANDLER_URL", "http://handler:8081"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@postgres:5432/annotations?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://redis:6379"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

// IsGateway returns true if the service is running as an API gateway.
func (c *Config) IsGateway() bool {
	return c.Role == "gateway"
}

// IsHandler returns true if the service is running as a handler.
func (c *Config) IsHandler() bool {
	return c.Role == "handler"
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
