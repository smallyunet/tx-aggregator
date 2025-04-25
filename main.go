package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tx-aggregator/consul"

	"github.com/gofiber/fiber/v2"
	consulapi "github.com/hashicorp/consul/api"

	"tx-aggregator/api"
	"tx-aggregator/cache"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/provider"
	"tx-aggregator/provider/ankr"
	"tx-aggregator/provider/blockscout"
	"tx-aggregator/router"
	"tx-aggregator/usecase/transaction"
	"tx-aggregator/utils"
)

func bootstrapPath() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	path := fmt.Sprintf("consul/bootstrap.%s.yaml", env)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return "consul/bootstrap.dev.yaml"
}

func main() {
	logger.Log.Info().Msg("==== Starting tx-aggregator ====")

	// 1. Load bootstrap config (for Consul + service registration)
	bootstrapFile := bootstrapPath()
	logger.Log.Info().Str("file", bootstrapFile).Msg("Loading bootstrap config")

	bootstrapCfg, err := consul.LoadBootstrap(bootstrapFile)
	if err != nil {
		logger.Log.Fatal().Err(err).Str("file", bootstrapFile).Msg("Failed to load bootstrap config")
	}
	logger.Log.Info().
		Str("consul.address", bootstrapCfg.Consul.Address).
		Str("consul.scheme", bootstrapCfg.Consul.Scheme).
		Str("consul.datacenter", bootstrapCfg.Consul.Datacenter).
		Str("service.name", bootstrapCfg.Service.Name).
		Str("service.ip", bootstrapCfg.Service.IP).
		Int("service.port", bootstrapCfg.Service.Port).
		Msg("Bootstrap config loaded")

	// 2. Load runtime config from Consul KV
	logger.Log.Info().Msg("Initializing runtime configuration from Consul KV")
	config.Init(bootstrapCfg)

	// 3. Init logger (after config)
	logger.Init(config.AppConfig.Log.Level)

	// 4. Setup Consul client
	logger.Log.Info().Str("consul.address", bootstrapCfg.Consul.Address).Msg("Creating Consul API client")
	consulCfg := consulapi.DefaultConfig()
	consulCfg.Address = bootstrapCfg.Consul.Address
	consulCfg.Scheme = bootstrapCfg.Consul.Scheme
	consulCfg.Datacenter = bootstrapCfg.Consul.Datacenter
	consulCfg.Token = bootstrapCfg.Consul.Token

	consulClient, err := consulapi.NewClient(consulCfg)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Failed to connect to Consul API")
	}
	logger.Log.Info().Msg("Connected to Consul successfully")

	// 5. Setup Redis
	logger.Log.Info().Strs("redis.addrs", config.AppConfig.Redis.Addrs).Msg("Initializing Redis cache")
	redisCache := cache.NewRedisCache(config.AppConfig.Redis.Addrs, config.AppConfig.Redis.Password)
	if redisCache == nil {
		logger.Log.Fatal().Msg("Failed to initialize Redis cache")
	}
	logger.Log.Info().Msg("Redis cache initialized")

	// 6. Setup providers
	logger.Log.Info().Msg("Setting up providers")
	registry := make(map[string]provider.Provider)
	registry["ankr"] = ankr.NewAnkrProvider(config.AppConfig.Ankr.APIKey, config.AppConfig.Ankr.URL)
	logger.Log.Info().Msg("Ankr provider registered")

	for _, bs := range config.AppConfig.Blockscout {
		chainID, err := utils.ChainIDByName(bs.ChainName)
		if err != nil {
			logger.Log.Warn().Str("chain", bs.ChainName).Msg("Invalid chain name, skipping Blockscout")
			continue
		}
		key := fmt.Sprintf("blockscout_%s", strings.ToLower(bs.ChainName))
		registry[key] = blockscout.NewBlockscoutProvider(chainID, bs)
		logger.Log.Info().Str("provider", key).Str("url", bs.URL).Msg("Blockscout provider registered")
	}

	multiProvider := provider.NewMultiProvider(registry)

	// 7. Setup Fiber app
	logger.Log.Info().Msg("Setting up HTTP server and routes")
	txService := transaction.NewService(redisCache, multiProvider)
	txHandler := api.NewTransactionHandler(txService)

	app := fiber.New()
	router.SetupRoutes(app, txHandler)

	// 8. Register service in Consul
	port := bootstrapCfg.Service.Port
	if port == 0 {
		port = config.AppConfig.Server.Port
	}
	serviceIP := bootstrapCfg.Service.IP
	if serviceIP == "" {
		serviceIP, _ = utils.GetLocalIPv4()
	}

	logger.Log.Info().
		Str("service.name", bootstrapCfg.Service.Name).
		Str("service.ip", serviceIP).
		Int("service.port", port).
		Msg("Registering service in Consul")

	deregister, err := consul.Register(consulClient, consul.Options{
		Name:       bootstrapCfg.Service.Name,
		ID:         fmt.Sprintf("%s-%d", bootstrapCfg.Service.Name, port),
		Address:    serviceIP,
		Port:       port,
		HealthPath: "/health",
		Meta:       map[string]string{"env": os.Getenv("APP_ENV")},
	})
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Consul service registration failed")
	}
	logger.Log.Info().Msg("Service registered successfully in Consul")

	// 9. Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Log.Warn().Str("signal", sig.String()).Msg("Received shutdown signal")

		if err := deregister(); err != nil {
			logger.Log.Error().Err(err).Msg("Failed to deregister from Consul")
		} else {
			logger.Log.Info().Msg("Deregistered from Consul successfully")
		}
		os.Exit(0)
	}()

	// 10. Start HTTP server
	logger.Log.Info().Int("port", port).Msg("Starting Fiber HTTP server")
	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		logger.Log.Fatal().Err(err).Msg("Fiber server terminated unexpectedly")
	}
}
