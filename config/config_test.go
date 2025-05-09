package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"tx-aggregator/types"
)

// TestInit_WithLocalFileOnly verifies that Init correctly loads configuration
// from environment variables and local defaults, without using Consul.
func TestInit_WithLocalFileOnly(t *testing.T) {
	// Reset viper and clear the global runtime config before each test.
	viper.Reset()
	SetCurrentConfig(types.Config{})

	// Simulate environment variables for a "test" environment
	// and override the server port.
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("APP_PORT", "9090")
	_ = os.Setenv("CONSUL_ADDR", "")       // disable Consul
	_ = os.Setenv("CONSUL_HTTP_TOKEN", "") // disable Consul auth
	defer os.Unsetenv("APP_ENV")
	defer os.Unsetenv("APP_PORT")
	defer os.Unsetenv("CONSUL_ADDR")
	defer os.Unsetenv("CONSUL_HTTP_TOKEN")

	// Build a minimal bootstrap config with an explicit Service.Name.
	bootstrap := &types.BootstrapConfig{
		Service: types.ServiceBootstrap{
			Name: "tx-aggregator",
		},
		Consul: types.ConsulBootstrap{
			Address: "",
			Token:   "",
		},
	}

	// Initialize the configuration and retrieve the current snapshot.
	Init(bootstrap)
	cfg := Current()

	// The APP_PORT environment variable should override server.port.
	assert.Equal(t, 9090, cfg.Server.Port, "APP_PORT should override server.port")

	// Since cache TTL default has been removed, Redis.TTLSeconds should be zero when not configured.
	assert.Equal(t, 0, cfg.Redis.TTLSeconds, "cache TTL should be zero when not configured")

	// Since log level default has been removed, Log.Level should be zero when not configured.
	assert.Equal(t, int8(0), cfg.Log.Level, "log level should be zero when not configured")

	// Since response max default has been removed, Response.Max should be zero when not configured.
	assert.Equal(t, 0, cfg.Response.Max, "response max should be zero when not configured")
}
