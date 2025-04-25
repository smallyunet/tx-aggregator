package utils

import (
	"net"
	"testing"
)

// TestGetLocalIPv4 tests the GetLocalIPv4 function for expected behavior.
func TestGetLocalIPv4(t *testing.T) {
	ip, err := GetLocalIPv4()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if ip == "" {
		t.Fatal("Expected a valid IP address, got empty string")
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		t.Fatalf("Returned IP is not a valid format: %s", ip)
	}

	if parsedIP.IsLoopback() {
		t.Fatalf("Returned IP should not be loopback, got: %s", ip)
	}

	t.Logf("Local IP address: %s", ip)
}
