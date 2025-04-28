package config

import (
	"os"
	"testing"
	"tx-aggregator/types"

	"github.com/stretchr/testify/assert"
)

// TestInit_WithLocalFileOnly tests that the configuration initializes correctly
// using environment variables and fallback defaults, without requiring Consul.
func TestInit_WithLocalFileOnly(t *testing.T) {
	// Set environment variables to simulate test environment
	_ = os.Setenv("APP_ENV", "test")       // Look for config/config.test.yaml
	_ = os.Setenv("APP_PORT", "9090")      // Override server port via env
	_ = os.Setenv("CONSUL_ADDR", "")       // Disable remote Consul loading
	_ = os.Setenv("CONSUL_HTTP_TOKEN", "") // Disable Consul auth

	defer os.Unsetenv("APP_ENV")
	defer os.Unsetenv("APP_PORT")
	defer os.Unsetenv("CONSUL_ADDR")
	defer os.Unsetenv("CONSUL_HTTP_TOKEN")

	// Create a dummy BootstrapConfig with no Consul settings
	bootstrap := &types.BootstrapConfig{
		Consul: types.ConsulBootstrap{
			Address: "",
			Token:   "",
		},
	}

	// Initialize configuration
	Init(bootstrap)

	// Assert that values are loaded correctly from env or default
	assert.Equal(t, 9090, AppConfig.Server.Port, "APP_PORT env should override default")
	assert.Equal(t, 60, AppConfig.Cache.TTLSeconds, "Default cache TTL should be 60")
	assert.Equal(t, int8(1), AppConfig.Log.Level, "Default log level should be 1")
	assert.Equal(t, 50, AppConfig.Response.Max, "Default response max should be 50")
}
