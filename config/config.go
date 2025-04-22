package config

import (
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"tx-aggregator/types"
)

// AppConfig is the global configuration instance
var AppConfig types.Config

func Init() {
	log.Println("Initializing configuration...")

	// Determine environment
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	log.Printf("Using APP_ENV: %s\n", env)

	// Set config file name based on environment
	viper.SetConfigName("config." + env) // config.dev.yaml / config.prod.yaml
	viper.SetConfigType("yaml")

	// Add config paths
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// Set default values
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("cache.ttl", 60)
	viper.SetDefault("log.level", 1)
	viper.SetDefault("response.max", 50)

	// Read configuration file
	log.Println("Loading configuration file...")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error loading config file: %w", err))
	}
	log.Printf("Configuration loaded from: %s\n", viper.ConfigFileUsed())

	// Unmarshal into struct
	log.Println("Unmarshaling configuration...")
	if err := viper.Unmarshal(&AppConfig); err != nil {
		panic(fmt.Errorf("unable to decode config into struct: %w", err))
	}
	log.Println("Configuration initialized successfully")
}
