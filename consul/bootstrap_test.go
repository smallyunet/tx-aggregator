package consul_test

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
	"tx-aggregator/consul"
	"tx-aggregator/types"
)

// helper: write a temporary bootstrap YAML file
func writeTempBootstrapFile(t *testing.T, content types.BootstrapConfig) string {
	t.Helper()
	tmpPath := filepath.Join(t.TempDir(), "bootstrap.yaml")

	data, err := yaml.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal YAML: %v", err)
	}

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	return tmpPath
}

func TestLoadBootstrap(t *testing.T) {
	// Step 1: Create a temporary YAML file
	defaultCfg := types.BootstrapConfig{
		Consul: types.ConsulBootstrap{
			Address:    "127.0.0.1:8500",
			Scheme:     "http",
			Datacenter: "dc1",
			Token:      "default-token",
		},
		Service: types.ServiceBootstrap{
			Name: "my-service",
			IP:   "192.168.1.100",
			Port: 8080,
		},
	}
	filePath := writeTempBootstrapFile(t, defaultCfg)

	// Step 2: Set environment variables to override some fields
	t.Setenv("CONSUL_ADDRESS", "10.0.0.1:9999")
	t.Setenv("CONSUL_SCHEME", "https")
	t.Setenv("SERVICE_IP", "10.10.10.10")
	t.Setenv("SERVICE_PORT", "9090")

	// Step 3: Load config using the target function
	cfg, err := consul.LoadBootstrap(filePath)
	if err != nil {
		t.Fatalf("LoadBootstrap() failed: %v", err)
	}

	// Step 4: Verify values from env override file
	if cfg.Consul.Address != "10.0.0.1:9999" {
		t.Errorf("Expected CONSUL_ADDRESS override, got %s", cfg.Consul.Address)
	}
	if cfg.Consul.Scheme != "https" {
		t.Errorf("Expected CONSUL_SCHEME override, got %s", cfg.Consul.Scheme)
	}
	if cfg.Service.IP != "10.10.10.10" {
		t.Errorf("Expected SERVICE_IP override, got %s", cfg.Service.IP)
	}
	if cfg.Service.Port != 9090 {
		t.Errorf("Expected SERVICE_PORT override, got %d", cfg.Service.Port)
	}
	if cfg.Service.Name != "my-service" {
		t.Errorf("Expected service name 'my-service', got %s", cfg.Service.Name)
	}
	if cfg.Consul.Datacenter != "dc1" {
		t.Errorf("Expected datacenter 'dc1', got %s", cfg.Consul.Datacenter)
	}
	if cfg.Consul.Token != "default-token" {
		t.Errorf("Expected token 'default-token', got %s", cfg.Consul.Token)
	}
}
