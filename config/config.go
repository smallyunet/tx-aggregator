package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// Config represents the application configuration structure
// It contains all the necessary configuration sections for the application
type Config struct {
	// Server configuration section
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`

	// Redis configuration section
	Redis struct {
		Addrs    []string `mapstructure:"addrs"`
		Password string   `mapstructure:"password"`
	} `mapstructure:"redis"`

	// Ankr API configuration section
	Ankr struct {
		APIKey             string   `mapstructure:"api_key"`
		URL                string   `mapstructure:"url"`
		RequestBlockchains []string `mapstructure:"request_blockchains"`
		RequestPageSize    int      `mapstructure:"request_page_size"`
	} `mapstructure:"ankr"`

	// Cache configuration section
	Cache struct {
		TTLSeconds int `mapstructure:"ttl"`
	} `mapstructure:"cache"`

	// Logging configuration section
	Log struct {
		Level int8 `mapstructure:"level"`
	} `mapstructure:"log"`

	// Response configuration section
	Response struct {
		Max int `mapstructure:"max"`
	} `mapstructure:"response"`

	// ChainIDs maps blockchain names to their respective chain IDs
	ChainIDs map[string]int64 `mapstructure:"chain_ids"`
}

// AppConfig is the global configuration instance
var AppConfig Config

// Init initializes the configuration by loading values from config file
// and setting default values for missing configurations
func Init() {
	log.Println("Initializing configuration...")

	// Set up viper configuration
	viper.SetConfigName("config")   // name of config file (without extension)
	viper.SetConfigType("yaml")     // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")        // optionally look for config in the working directory
	viper.AddConfigPath("./config") // or in a config subdirectory

	// Set default values for configuration
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("cache.ttl", 60)
	viper.SetDefault("log.level", 1)
	viper.SetDefault("response.max", 50)

	log.Println("Loading configuration file...")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error loading config file: %w", err))
	}
	log.Printf("Configuration file loaded from: %s\n", viper.ConfigFileUsed())

	log.Println("Unmarshaling configuration...")
	if err := viper.Unmarshal(&AppConfig); err != nil {
		panic(fmt.Errorf("unable to decode config into struct: %w", err))
	}
	log.Println("Configuration initialized successfully")
}
