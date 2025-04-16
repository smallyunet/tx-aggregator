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
	logger.Log.Info().Msg("Redis cache client initialized successfully")

	// Initialize Ankr provider
	logger.Log.Info().Msg("Initializing Ankr provider...")
	ankrProvider := provider.NewAnkrProvider(config.AppConfig.Ankr.APIKey, config.AppConfig.Ankr.URL)
	logger.Log.Info().Msg("Ankr provider initialized successfully")

	// Initialize multi-provider with Ankr as the primary provider
	logger.Log.Info().Msg("Initializing multi-provider...")
	multiProvider := provider.NewMultiProvider(ankrProvider)
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
