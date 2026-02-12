package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_DefaultValues(t *testing.T) {
	// Clear environment variables to test defaults
	originalRole := os.Getenv("SERVICE_ROLE")
	originalPort := os.Getenv("SERVER_PORT")
	originalEnv := os.Getenv("ENVIRONMENT")
	defer func() {
		os.Setenv("SERVICE_ROLE", originalRole)
		os.Setenv("SERVER_PORT", originalPort)
		os.Setenv("ENVIRONMENT", originalEnv)
	}()

	os.Unsetenv("SERVICE_ROLE")
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("ENVIRONMENT")

	cfg := New()

	assert.Equal(t, "gateway", cfg.Role)
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, "http://handler:8081", cfg.HandlerURL)
}

func TestNew_EnvironmentOverrides(t *testing.T) {
	originalRole := os.Getenv("SERVICE_ROLE")
	originalPort := os.Getenv("SERVER_PORT")
	originalEnv := os.Getenv("ENVIRONMENT")
	defer func() {
		os.Setenv("SERVICE_ROLE", originalRole)
		os.Setenv("SERVER_PORT", originalPort)
		os.Setenv("ENVIRONMENT", originalEnv)
	}()

	os.Setenv("SERVICE_ROLE", "handler")
	os.Setenv("SERVER_PORT", "9000")
	os.Setenv("ENVIRONMENT", "production")

	cfg := New()

	assert.Equal(t, "handler", cfg.Role)
	assert.Equal(t, "9000", cfg.ServerPort)
	assert.Equal(t, "production", cfg.Environment)
}

func TestIsGateway(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"gateway", true},
		{"handler", false},
		{"other", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			cfg := &Config{Role: tt.role}
			assert.Equal(t, tt.expected, cfg.IsGateway())
		})
	}
}

func TestIsHandler(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"gateway", false},
		{"handler", true},
		{"other", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			cfg := &Config{Role: tt.role}
			assert.Equal(t, tt.expected, cfg.IsHandler())
		})
	}
}

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		env      string
		expected bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Environment: tt.env}
			assert.Equal(t, tt.expected, cfg.IsDevelopment())
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Test with existing env var
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	result := getEnv("TEST_VAR", "default")
	assert.Equal(t, "test_value", result)

	// Test with non-existing env var
	result = getEnv("NON_EXISTING_VAR", "default_value")
	assert.Equal(t, "default_value", result)
}

func TestGetEnvInt(t *testing.T) {
	// Test with valid integer
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	result := getEnvInt("TEST_INT", 10)
	assert.Equal(t, 42, result)

	// Test with invalid integer
	os.Setenv("TEST_INVALID_INT", "not_a_number")
	defer os.Unsetenv("TEST_INVALID_INT")

	result = getEnvInt("TEST_INVALID_INT", 10)
	assert.Equal(t, 10, result)

	// Test with non-existing env var
	result = getEnvInt("NON_EXISTING_INT", 100)
	assert.Equal(t, 100, result)
}
