// main.go
package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tx-aggregator/consul"

	consulapi "github.com/hashicorp/consul/api"

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
	"tx-aggregator/utils"
)

func bootstrapPath() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	// search order: consul/bootstrap.<env>.yaml → consul/bootstrap.test2.yaml
	candidate := fmt.Sprintf("consul/bootstrap.%s.yaml", env)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return "consul/bootstrap.dev.yaml" // fallback
}

func main() {
	//----------------------------------------------------------------------
	// 1. Load application config and initialize logger
	//----------------------------------------------------------------------
	config.Init()
	logger.Init()

	path := bootstrapPath()
	bootstrapCfg, err := consul.LoadBootstrap(path)
	if err != nil {
		logger.Log.Fatal().Err(err).Str("path", path).Msg("Failed to load bootstrap config")
	}
	logger.Log.Info().Str("bootstrap", path).Msg("loaded bootstrap configuration")

	//----------------------------------------------------------------------
	// 2. Build Consul client from bootstrap parameters
	//----------------------------------------------------------------------
	consulCfg := consulapi.DefaultConfig()
	consulCfg.Address = bootstrapCfg.Consul.Address
	consulCfg.Scheme = bootstrapCfg.Consul.Scheme
	consulCfg.Datacenter = bootstrapCfg.Consul.Datacenter
	consulCfg.Token = bootstrapCfg.Consul.Token

	consulClient, err := consulapi.NewClient(consulCfg)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("unable to connect to Consul")
	}

	//----------------------------------------------------------------------
	// 3. Initialize Redis cache (mandatory for the service layer)
	//----------------------------------------------------------------------
	redisCache := cache.NewRedisCache(config.AppConfig.Redis.Addrs, config.AppConfig.Redis.Password)
	if redisCache == nil {
		logger.Log.Fatal().Msg("unable to create Redis cache client")
	}

	//----------------------------------------------------------------------
	// 4. Provider registry (Ankr, Blockscout, QuickNode, etc.)
	//----------------------------------------------------------------------
	registry := make(map[string]provider.Provider)

	ankrProvider := ankr.NewAnkrProvider(
		config.AppConfig.Ankr.APIKey,
		config.AppConfig.Ankr.URL,
	)
	registry["ankr"] = ankrProvider

	for _, bs := range config.AppConfig.Blockscout {
		chainID, err := utils.ChainIDByName(bs.ChainName)
		if err != nil {
			logger.Log.Warn().Str("chain", bs.ChainName).Msg("unknown chain; skip Blockscout provider")
			continue
		}
		key := fmt.Sprintf("blockscout_%s", strings.ToLower(bs.ChainName))
		registry[key] = blockscout.NewBlockscoutProvider(chainID, bs)
	}

	multiProvider := provider.NewMultiProvider(registry)

	//----------------------------------------------------------------------
	// 5. Build service layer, HTTP handlers, and Fiber app
	//----------------------------------------------------------------------
	txService := transaction.NewService(redisCache, multiProvider)
	txHandler := api.NewTransactionHandler(txService)

	app := fiber.New()
	router.SetupRoutes(app, txHandler)

	//----------------------------------------------------------------------
	// 6. Register the service in Consul and set up deregistration on exit
	//----------------------------------------------------------------------
	port := config.AppConfig.Server.Port

	serviceIP := bootstrapCfg.Service.IP
	if serviceIP == "" {
		serviceIP, _ = utils.GetLocalIPv4() // fallback to detected IP
	}

	logger.Log.Info().
		Str("service", bootstrapCfg.Service.Name).
		Str("ip", serviceIP).
		Int("port", port).
		Msg("service registered in Consul")

	deregister, err := consul.Register(consulClient, consul.Options{
		Name:       bootstrapCfg.Service.Name, // e.g. "tx-aggregator"
		ID:         fmt.Sprintf("%s-%d", bootstrapCfg.Service.Name, port),
		Address:    serviceIP,
		Port:       port,
		HealthPath: "/health",
		Meta:       map[string]string{"env": os.Getenv("APP_ENV")},
	})
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Consul registration failed")
	}

	// Handle SIGINT/SIGTERM for graceful shutdown and Consul deregistration
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		if err := deregister(); err != nil {
			logger.Log.Error().Err(err).Msg("Consul deregistration failed")
		}
		os.Exit(0)
	}()

	//----------------------------------------------------------------------
	// 7. Start Fiber HTTP server
	//----------------------------------------------------------------------
	if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
		logger.Log.Fatal().Err(err).Msg("Fiber server terminated")
	}
}
