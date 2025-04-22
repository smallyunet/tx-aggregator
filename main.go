package main

import (
	"fmt"
	"os"
	"tx-aggregator/utils"

	"github.com/gofiber/fiber/v2"

	"tx-aggregator/api"
	"tx-aggregator/cache"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/provider"
	"tx-aggregator/provider/ankr"
	"tx-aggregator/provider/blockscout"
	"tx-aggregator/router"
	"tx-aggregator/usecase/transaction"
)

func main() {
	// Step 1: Load configuration
	config.Init()
	logger.Log.Info().Msg("Configuration loaded")

	// Step 2: Initialize logger
	logger.Init()
	logger.Log.Info().Msg("Logger initialized")

	// Step 3: Initialize Redis
	redisCache := cache.NewRedisCache(config.AppConfig.Redis.Addrs, config.AppConfig.Redis.Password)
	if redisCache == nil {
		logger.Log.Fatal().Msg("Failed to initialize Redis cache client")
	}
	logger.Log.Info().
		Strs("redis_addresses", config.AppConfig.Redis.Addrs).
		Msg("Redis cache initialized")

	// Step 4: Initialize data providers
	var providers []provider.Provider

	// Ankr provider
	logger.Log.Info().Str("ankr_url", config.AppConfig.Ankr.URL).Msg("Adding Ankr provider")
	providers = append(providers, ankr.NewAnkrProvider(config.AppConfig.Ankr.APIKey, config.AppConfig.Ankr.URL))

	// Blockscout providers
	for _, bs := range config.AppConfig.Blockscout {
		chainID, err := utils.ChainIDByName(bs.ChainName)
		if err != nil {
			logger.Log.Warn().Str("chain_name", bs.ChainName).Msg("Unknown chain name, skipping Blockscout provider")
			continue
		}
		logger.Log.Info().
			Str("chain_name", bs.ChainName).
			Str("blockscout_url", bs.URL).
			Msg("Adding Blockscout provider")
		providers = append(providers, blockscout.NewBlockscoutProvider(chainID, bs))
	}

	multiProvider := provider.NewMultiProvider(providers...)
	if multiProvider == nil {
		logger.Log.Fatal().Msg("Failed to initialize multi-provider")
	}
	logger.Log.Info().Msg("Multi-provider initialized")

	// Step 5: Initialize usecase service
	txService := transaction.NewService(redisCache, multiProvider)

	// Step 6: Initialize handler
	txHandler := api.NewTransactionHandler(txService)

	// Step 7: Create HTTP server
	app := fiber.New()
	router.SetupRoutes(app, txHandler)
	logger.Log.Info().Msg("Routes configured")

	// Step 8: Start server
	port := config.AppConfig.Server.Port
	logger.Log.Info().Int("port", port).Msg("Starting HTTP server")

	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		logger.Log.Error().Err(err).Msg("Failed to start server")
		os.Exit(1)
	}
}
