package consul

import (
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
)

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

// Register registers the service to the local Consul Agent.
// Returns: a deregistration function (can be called on exit) and a possible error.
func Register(client *api.Client, opt Options) (func() error, error) {
	if opt.Interval == 0 {
		opt.Interval = 10 * time.Second
	}
	if opt.Timeout == 0 {
		opt.Timeout = 1 * time.Second
	}
	if opt.Deregister == 0 {
		opt.Deregister = 5 * time.Minute
	}

	reg := &api.AgentServiceRegistration{
		ID:      opt.ID,
		Name:    opt.Name,
		Tags:    opt.Tags,
		Port:    opt.Port,
		Address: opt.Address,
		Meta:    opt.Meta,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d%s", opt.Address, opt.Port, opt.HealthPath),
			Interval:                       opt.Interval.String(),
			Timeout:                        opt.Timeout.String(),
			DeregisterCriticalServiceAfter: opt.Deregister.String(),
		},
	}

	if err := client.Agent().ServiceRegister(reg); err != nil {
		return nil, fmt.Errorf("consul register: %w", err)
	}

	// Return the deregistration function
	return func() error {
		return client.Agent().ServiceDeregister(opt.ID)
	}, nil
}
