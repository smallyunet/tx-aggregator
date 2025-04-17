package config

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

// Config represents the application configuration structure
type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`

	Redis struct {
		Addrs    []string `mapstructure:"addrs"`
		Password string   `mapstructure:"password"`
	} `mapstructure:"redis"`

	Ankr struct {
		APIKey             string   `mapstructure:"api_key"`
		URL                string   `mapstructure:"url"`
		RequestBlockchains []string `mapstructure:"request_blockchains"`
		RequestPageSize    int      `mapstructure:"request_page_size"`
	} `mapstructure:"ankr"`

	Blockscout []struct {
		URL       string `mapstructure:"url"`
		ChainName string `mapstructure:"chain_name"`
		RPCURL    string `mapstructure:"rpc_url"`
	} `mapstructure:"blockscout"`

	Cache struct {
		TTLSeconds int `mapstructure:"ttl"`
	} `mapstructure:"cache"`

	Log struct {
		Level int8 `mapstructure:"level"`
	} `mapstructure:"log"`

	Response struct {
		Max int `mapstructure:"max"`
	} `mapstructure:"response"`

	ChainIDs map[string]int64 `mapstructure:"chain_ids"`
}

// AppConfig is the global configuration instance
var AppConfig Config

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

// ChainIDByName returns the chain ID for a given chain name.
// If the chain name is not found in the configuration, it returns 0.
// This function is used to map chain names to their corresponding IDs
// as defined in the application configuration.
func ChainIDByName(name string) int64 {
	if id, ok := AppConfig.ChainIDs[name]; ok {
		return id
	}
	return 0
}
