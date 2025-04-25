// Package config handles loading runtime configuration from Consul KV
package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote" // enables remote KV support
	"tx-aggregator/logger"
	"tx-aggregator/types"
)

var AppConfig types.Config

// Init loads application configuration from Consul KV, given a BootstrapConfig.
func Init(bootstrap *types.BootstrapConfig) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	consulAddr := bootstrap.Consul.Address
	consulToken := bootstrap.Consul.Token

	// Override from env if present
	if consulAddr == "" {
		consulAddr = os.Getenv("CONSUL_ADDR")
	}
	if consulToken == "" {
		consulToken = os.Getenv("CONSUL_HTTP_TOKEN")
	}
	if consulToken != "" {
		_ = os.Setenv("CONSUL_HTTP_TOKEN", consulToken)
	}

	logger.Log.Info().
		Str("env", env).
		Str("consul.address", consulAddr).
		Str("consul.token", maskToken(consulToken)).
		Msg("Initializing configuration")

	// Load from Consul KV
	consulKey := "config/tx-aggregator/" + env
	remoteOK := false

	if consulAddr != "" {
		if err := viper.AddRemoteProvider("consul", consulAddr, consulKey); err != nil {
			logger.Log.Warn().Err(err).Str("provider", consulAddr).Msg("Failed to register remote config provider")
		} else {
			viper.SetConfigType("yaml")
			if err := viper.ReadRemoteConfig(); err != nil {
				logger.Log.Warn().Err(err).Str("key", consulKey).Msg("Failed to read remote config, falling back to local/defaults")
			} else {
				remoteOK = true
				logger.Log.Info().
					Str("consul", consulAddr).
					Str("key", consulKey).
					Msg("Loaded configuration from Consul KV")
			}
		}
	}

	// Optional local override
	overrideFile := "config." + env + ".yaml"
	logger.Log.Info().Str("override", overrideFile).Msg("Merging local override config if exists")
	viper.SetConfigName("config." + env)
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	_ = viper.MergeInConfig() // optional

	// Set defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("cache.ttl", 60)
	viper.SetDefault("log.level", 1)
	viper.SetDefault("response.max", 50)

	_ = viper.BindEnv("server.port", "APP_PORT")

	// Unmarshal into struct
	if err := viper.Unmarshal(&AppConfig); err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to unmarshal configuration")
	}

	logger.Log.Info().
		Str("env", env).
		Int("port", AppConfig.Server.Port).
		Msg("Configuration initialized successfully")

	// Watch for changes (only if remote config succeeded)
	if remoteOK {
		logger.Log.Info().Msg("Starting remote config watcher")
		go func() {
			for {
				if err := viper.WatchRemoteConfig(); err != nil {
					logger.Log.Error().Err(err).Msg("Remote config watch failed")
				}
				var updated types.Config
				if err := viper.Unmarshal(&updated); err == nil {
					AppConfig = updated
					logger.Log.Info().Msg("Reloaded configuration from Consul KV")
				}
				time.Sleep(10 * time.Second)
			}
		}()
	}
}

// maskToken hides the middle part of the token string for safe logging
func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
