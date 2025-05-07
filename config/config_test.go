package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"tx-aggregator/types"
)

// TestInit_WithLocalFileOnly verifies that configuration initializes correctly
// using environment variables and local defaults, without connecting to Consul.
func TestInit_WithLocalFileOnly(t *testing.T) {
	// Set environment variables to simulate a test environment
	_ = os.Setenv("APP_ENV", "test")       // Should look for config/config.test.yaml
	_ = os.Setenv("APP_PORT", "9090")      // Should override server port via env
	_ = os.Setenv("CONSUL_ADDR", "")       // Disable Consul loading
	_ = os.Setenv("CONSUL_HTTP_TOKEN", "") // Disable Consul auth

	defer os.Unsetenv("APP_ENV")
	defer os.Unsetenv("APP_PORT")
	defer os.Unsetenv("CONSUL_ADDR")
	defer os.Unsetenv("CONSUL_HTTP_TOKEN")

	// Create a minimal bootstrap configuration (Consul disabled)
	bootstrap := &types.BootstrapConfig{
		Consul: types.ConsulBootstrap{
			Address: "",
			Token:   "",
		},
	}

	// Run the initialization logic
	Init(bootstrap)

	// Fetch the current config snapshot
	cfg := Current()

	// Assert that configuration values are correctly set
	assert.Equal(t, 9090, cfg.Server.Port, "APP_PORT environment variable should override default")
	assert.Equal(t, 0, cfg.Redis.TTLSeconds, "Default cache TTL should be 60")
	assert.Equal(t, int8(1), cfg.Log.Level, "Default log level should be 1")
	assert.Equal(t, 50, cfg.Response.Max, "Default max response size should be 50")
}
