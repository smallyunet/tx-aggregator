package consul

import (
	"fmt"

	"github.com/spf13/viper"
)

// ------------------------------------------------------------
// Data structures — corresponding to bootstrap.test2.yaml
// ------------------------------------------------------------

// consul: node
type ConsulBootstrap struct {
	Address    string `yaml:"address"`    // 10.234.99.5:8500
	Scheme     string `yaml:"scheme"`     // http / https
	Datacenter string `yaml:"datacenter"` // dc1…
	Token      string `yaml:"token"`      // ACL token, can be empty
}

// service: node
type ServiceBootstrap struct {
	Name string `yaml:"name"` // tx-aggregator
	IP   string `yaml:"ip"`   // Leave empty to detect local IP at runtime
	Port int    `yaml:"port"` // 0 = use runtime Server.Port
}

// BootstrapConfig top-level structure
type BootstrapConfig struct {
	Consul  ConsulBootstrap  `yaml:"consul"`
	Service ServiceBootstrap `yaml:"service"`
}

// ------------------------------------------------------------
// Read function
// ------------------------------------------------------------

// LoadBootstrap reads and parses bootstrap.test2.yaml, returning the filled struct.
// path can be an absolute path or a relative path (e.g., `config/bootstrap.test2.yaml`).
func LoadBootstrap(path string) (*BootstrapConfig, error) {
	v := viper.New()
	v.SetConfigFile(path) // Specify the file
	v.SetConfigType("yaml")

	// 1) Read the file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read bootstrap config: %w", err)
	}

	// 2) Deserialize
	var cfg BootstrapConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal bootstrap config: %w", err)
	}

	return &cfg, nil
}
