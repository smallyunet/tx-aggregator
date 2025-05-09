// Package config handles loading and hot‑reloading runtime configuration
// from Consul KV (with an optional local‑file override).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote" // enables Consul/etcd remote KV support

	"tx-aggregator/logger"
	"tx-aggregator/types"
)

// runtimeCfg always holds the newest configuration.
// atomic.Value gives us cheap, lock‑free, thread‑safe reads.
var runtimeCfg atomic.Value // stores types.Config

// Current returns a read‑only snapshot of the latest configuration.
func Current() types.Config {
	v := runtimeCfg.Load()
	if v == nil {
		return types.Config{} // zero until Init succeeds
	}
	return v.(types.Config)
}

// Init loads configuration from Consul KV (plus optional local overrides)
// and starts a background goroutine that refreshes the settings every 10 s.
func Init(bootstrap *types.BootstrapConfig) {
	/* ────────────────────────────────────────────────────────────────
	   1. Resolve environment, Consul address & token
	---------------------------------------------------------------- */
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	consulAddr := firstNonEmpty(
		bootstrap.Consul.Address,
		os.Getenv("CONSUL_ADDR"),
	)

	consulToken := firstNonEmpty(
		bootstrap.Consul.Token,
		os.Getenv("CONSUL_HTTP_TOKEN"),
	)
	if consulToken != "" {
		_ = os.Setenv("CONSUL_HTTP_TOKEN", consulToken) // for the Consul client
	}

	logger.Log.Info().
		Str("env", env).
		Str("consul.address", consulAddr).
		Str("consul.token", maskToken(consulToken)).
		Msg("initialising configuration")

	/* ────────────────────────────────────────────────────────────────
	   2. Default values (safest, lowest precedence)
	---------------------------------------------------------------- */
	viper.SetDefault("server.port", 8080)
	_ = viper.BindEnv("server.port", "APP_PORT")

	/* ────────────────────────────────────────────────────────────────
	   3. Optional local override  (medium precedence)
	---------------------------------------------------------------- */
	overrideFile := fmt.Sprintf("config.%s.yaml", env)
	viper.SetConfigName(strings.TrimSuffix(overrideFile, ".yaml"))
	viper.AddConfigPath(filepath.Join(".", types.ConfigFolderPath)) // e.g. ./config
	viper.AddConfigPath(".")                                        // project root

	_ = viper.MergeInConfig() // ignore 'file not found'; merge if present

	/* ────────────────────────────────────────────────────────────────
	   4. Load Consul KV  (highest precedence)
	---------------------------------------------------------------- */
	key := fmt.Sprintf("config/%s/%s", bootstrap.Service.Name, env) // KV path containing the YAML blob
	if consulAddr != "" {
		if err := viper.AddRemoteProvider("consul", consulAddr, key); err != nil {
			logger.Log.Fatal().Err(err).Msg("cannot register Consul remote provider")
		}
		viper.SetConfigType("yaml")

		if err := viper.ReadRemoteConfig(); err != nil {
			logger.Log.Fatal().Err(err).Msg("cannot read configuration from Consul KV")
		}
	} else {
		logger.Log.Warn().Msg("CONSUL_ADDR missing – falling back to local defaults only")
	}

	/* ────────────────────────────────────────────────────────────────
	   5. Unmarshal first snapshot & publish it
	---------------------------------------------------------------- */
	var cfg types.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		logger.Log.Fatal().Err(err).Msg("cannot unmarshal initial configuration")
	}
	runtimeCfg.Store(cfg) // first snapshot

	logger.Log.Info().
		Int("server.port", cfg.Server.Port).
		Msg("configuration loaded")

	/* ────────────────────────────────────────────────────────────────
	   6. Background refresher – poll Consul every 10 s
	---------------------------------------------------------------- */
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			/* 1. create a clean viper instance and pull the KV blob */
			remote := viper.New()
			remote.SetConfigType("yaml")
			if err := remote.AddRemoteProvider("consul", consulAddr, key); err != nil {
				logger.Log.Error().Err(err).Msg("consul provider init failed")
				continue
			}
			if err := remote.ReadRemoteConfig(); err != nil {
				logger.Log.Error().Err(err).Msg("cannot fetch remote config")
				continue
			}

			/* 2. unmarshal into a concrete struct */
			var updated types.Config
			if err := remote.Unmarshal(&updated); err != nil {
				logger.Log.Error().Err(err).Msg("unmarshal failed")
				continue
			}

			/* 3. swap in only when something actually changed */
			if !reflect.DeepEqual(Current(), updated) {
				runtimeCfg.Store(updated)
				logger.Log.Info().Msg("configuration hot‑reloaded from Consul KV")
			}
		}
	}()
}

/* ──────────────────────────────────────────────────────────────────
   Helpers
-------------------------------------------------------------------*/

// maskToken hides the middle part of a Consul token when logging.
func maskToken(t string) string {
	if len(t) <= 8 {
		return "********"
	}
	return t[:4] + "****" + t[len(t)-4:]
}

// firstNonEmpty returns the first argument that is not "".
func firstNonEmpty(candidates ...string) string {
	for _, c := range candidates {
		if c != "" {
			return c
		}
	}
	return ""
}

// SetCurrentConfig is for testing purposes only.
func SetCurrentConfig(cfg types.Config) {
	runtimeCfg.Store(cfg)
}
