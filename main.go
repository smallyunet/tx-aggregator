package main

import (
	"fmt"
	"os"

	"tx-aggregator/api"
	"tx-aggregator/cache"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/provider"
	"tx-aggregator/router"

	"github.com/gofiber/fiber/v2"
)

// main is the entry point of the application
// It initializes all necessary components and starts the HTTP server
func main() {
	// Initialize configuration
	logger.Log.Info().Msg("Loading configuration...")
	config.Init()
	logger.Log.Info().Msg("Configuration loaded successfully")

	// Initialize logger
	logger.Log.Info().Msg("Initializing logger...")
	logger.Init()
	logger.Log.Info().Msg("Logger initialized successfully")

	// Initialize Redis cache client
	logger.Log.Info().Msg("Initializing Redis cache client...")
	redisCache := cache.NewRedisCache(config.AppConfig.Redis.Addrs, config.AppConfig.Redis.Password)
	if redisCache == nil {
		logger.Log.Fatal().Msg("Failed to initialize Redis cache client")
	}
	logger.Log.Info().
		Strs("redis_addresses", config.AppConfig.Redis.Addrs).
		Msg("Redis cache client initialized successfully")

	// Initialize datasource providers
	var providers []provider.Provider

	// Ankr
	logger.Log.Info().
		Str("ankr_url", config.AppConfig.Ankr.URL).
		Msg("Adding Ankr provider...")
	providers = append(providers, provider.NewAnkrProvider(config.AppConfig.Ankr.APIKey, config.AppConfig.Ankr.URL))
	logger.Log.Info().Msg("Ankr provider added successfully")

	// Blockscout
	logger.Log.Info().Msg("Adding Blockscout providers...")
	for _, bs := range config.AppConfig.Blockscout {
		chainID, exists := config.AppConfig.ChainIDs[bs.ChainName]
		if !exists {
			logger.Log.Warn().
				Str("chain_name", bs.ChainName).
				Msg("Chain ID not found for Blockscout provider, skipping...")
			continue
		}
		logger.Log.Info().
			Str("blockscout_url", bs.URL).
			Str("chain_name", bs.ChainName).
			Int64("chain_id", chainID).
			Msg("Adding Blockscout provider...")
		providers = append(providers, provider.NewBlockscoutProvider(bs.URL, chainID, bs.ChainName))
		logger.Log.Info().
			Str("chain_name", bs.ChainName).
			Msg("Blockscout provider added successfully")
	}

	// Initialize multi-provider with Ankr as the primary provider
	logger.Log.Info().Msg("Initializing multi-provider...")
	multiProvider := provider.NewMultiProvider(providers...)
	if multiProvider == nil {
		logger.Log.Fatal().Msg("Failed to initialize multi-provider")
	}
	logger.Log.Info().Msg("Multi-provider initialized successfully")
	// Initialize API components
	logger.Log.Info().Msg("Initializing API components...")
	api.Init(multiProvider, redisCache)
	logger.Log.Info().Msg("API components initialized successfully")

	// Create new Fiber application
	logger.Log.Info().Msg("Creating new Fiber application...")
	app := fiber.New()

	// Setup routes
	logger.Log.Info().Msg("Setting up routes...")
	router.SetupRoutes(app)
	logger.Log.Info().Msg("Routes setup completed successfully")

	// Start the server
	port := config.AppConfig.Server.Port
	logger.Log.Info().
		Str("port", port).
		Msg("Starting server...")

	if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to start server")
		os.Exit(1)
	}
}
