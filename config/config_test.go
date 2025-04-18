package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInit ensures that Init sets default values and loads configuration properly.
func TestInit(t *testing.T) {
	// Set an environment variable to simulate a different environment if needed.
	os.Setenv("APP_ENV", "dev")
	defer os.Unsetenv("APP_ENV")

	// Call the Init function from the config package.
	Init()

	// Verify that default values or loaded values match expectations.
	assert.Equal(t, 8080, AppConfig.Server.Port)
	assert.Equal(t, 60, AppConfig.Cache.TTLSeconds)
	assert.Equal(t, int8(0), AppConfig.Log.Level)
	assert.Equal(t, 50, AppConfig.Response.Max)
}
