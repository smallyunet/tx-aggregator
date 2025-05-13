package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"tx-aggregator/types"
)

func TestInit_WithLocalFileOnly(t *testing.T) {
	// Clean state
	viper.Reset()
	SetCurrentConfig(types.Config{})

	// Set env variables to simulate local only config
	_ = os.Setenv("APP_ENV", "test")
	_ = os.Setenv("APP_PORT", "9090")
	_ = os.Setenv("CONSUL_ADDR", "")       // ensure Consul is disabled
	_ = os.Setenv("CONSUL_HTTP_TOKEN", "") // ensure Consul token is empty

	defer func() {
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("APP_PORT")
		_ = os.Unsetenv("CONSUL_ADDR")
		_ = os.Unsetenv("CONSUL_HTTP_TOKEN")
	}()

	// Bootstrap config with minimal valid fields
	bootstrap := &types.BootstrapConfig{
		Service: types.ServiceBootstrap{
			Name: "tx-aggregator",
		},
		Consul: types.ConsulBootstrap{
			Address: "",
			Token:   "",
		},
	}

	// Execute init (should only load local and env, no Consul)
	Init(bootstrap)
	cfg := Current()

	// ✅ Assert that server.port picked from env APP_PORT
	assert.Equal(t, 9090, cfg.Server.Port, "server.port should match APP_PORT env")

	// ✅ Assert Redis TTL fallback to zero if not present
	assert.Zero(t, cfg.Redis.TTLSeconds, "Redis.TTLSeconds should be zero if not configured")

	// ✅ Assert log level fallback to zero if not present
	assert.Zero(t, cfg.Log.Level, "Log.Level should be zero if not configured")

	// ✅ Assert response max fallback to zero if not present
	assert.Zero(t, cfg.Response.Max, "Response.Max should be zero if not configured")
}
