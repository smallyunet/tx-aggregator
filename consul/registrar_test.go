package consul_test

import (
	"testing"
	"tx-aggregator/consul"
	"tx-aggregator/types"

	"github.com/hashicorp/consul/api"
)

func TestRegister(t *testing.T) {
	// Create a new Consul API client (connects to default local agent)
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create Consul client: %v", err)
	}

	// Define the registration options
	opts := types.Options{
		Name:       "test-service",
		ID:         "test-service-1234",
		Address:    "127.0.0.1",
		Port:       8080,
		Tags:       []string{"test", "unit"},
		Meta:       map[string]string{"env": "test"},
		HealthPath: "/health",
	}

	// Call Register
	deregister, err := consul.Register(client, opts)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}
	t.Log("Service registered successfully")

	// Ensure the service appears in the Consul catalog
	svc, _, err := client.Agent().Service(opts.ID, nil)
	if err != nil {
		t.Fatalf("Failed to fetch registered service: %v", err)
	}
	if svc.Service != opts.Name {
		t.Errorf("Expected service name %s, got %s", opts.Name, svc.Service)
	}

	// Call the returned deregistration function
	if err := deregister(); err != nil {
		t.Fatalf("Deregister() failed: %v", err)
	}
	t.Log("Service deregistered successfully")
}
