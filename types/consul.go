package types

import "time"

// ConsulBootstrap holds configuration for connecting to a Consul agent.
type ConsulBootstrap struct {
	Address    string `yaml:"address"`    // e.g., "10.234.99.5:8500"
	Scheme     string `yaml:"scheme"`     // "http" or "https"
	Datacenter string `yaml:"datacenter"` // Consul datacenter
	Token      string `yaml:"token"`      // ACL token (optional)
}

// ServiceBootstrap holds metadata about the current service.
type ServiceBootstrap struct {
	Name string `yaml:"name"` // e.g., "tx-aggregator"
	IP   string `yaml:"ip"`   // Service IP; if empty, detect dynamically
	Port int    `yaml:"port"` // Service port; 0 means use runtime port
}

// BootstrapConfig is the root structure for the bootstrap configuration file.
type BootstrapConfig struct {
	Consul  ConsulBootstrap  `yaml:"consul"`
	Service ServiceBootstrap `yaml:"service"`
}

// Options describes all the dynamic information required for a service registration.
type Options struct {
	Name       string            // Service name, e.g., "tx-aggregator"
	ID         string            // Unique instance ID, recommended format: Name-PORT or Name-UUID
	Address    string            // Host IP
	Port       int               // Service listening port
	Tags       []string          // Optional: Consul Tags
	Meta       map[string]string // Optional: Metadata
	HealthPath string            // Health check HTTP path, e.g., "/health"
	Interval   time.Duration     // Check interval
	Timeout    time.Duration     // Timeout
	Deregister time.Duration     // Automatically deregister after continuous failures
}
