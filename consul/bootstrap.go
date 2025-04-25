package consul

import (
	"fmt"
	"os"
	"path/filepath"
	"tx-aggregator/types"

	"github.com/spf13/viper"
)

// BootstrapPath returns the full path to the bootstrap YAML config file.
// It checks for a file named bootstrap.<env>.yaml under types.ConfigFolderPath.
// Falls back to bootstrap.dev.yaml if specific file is not found.
func BootstrapPath() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	candidate := filepath.Join(types.ConfigFolderPath, fmt.Sprintf("bootstrap.%s.yaml", env))
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	return filepath.Join(types.ConfigFolderPath, "bootstrap.dev.yaml")
}

// LoadBootstrap reads and parses the bootstrap YAML configuration file.
// It allows overrides from environment variables.
// Path can be relative or absolute, e.g., "config/bootstrap.yaml".
func LoadBootstrap(path string) (*types.BootstrapConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Step 1: Read the file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read bootstrap config: %w", err)
	}

	// Step 2: Unmarshal into struct
	var cfg types.BootstrapConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal bootstrap config: %w", err)
	}

	// Step 3: Override fields with environment variables if present
	if env := os.Getenv("CONSUL_ADDRESS"); env != "" {
		cfg.Consul.Address = env
	}
	if env := os.Getenv("CONSUL_SCHEME"); env != "" {
		cfg.Consul.Scheme = env
	}
	if env := os.Getenv("CONSUL_TOKEN"); env != "" {
		cfg.Consul.Token = env
	}
	if env := os.Getenv("SERVICE_IP"); env != "" {
		cfg.Service.IP = env
	}
	if env := os.Getenv("SERVICE_PORT"); env != "" {
		// Optionally parse string to int if needed
		var port int
		if _, err := fmt.Sscanf(env, "%d", &port); err == nil {
			cfg.Service.Port = port
		}
	}

	return &cfg, nil
}
