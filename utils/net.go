package utils

import (
	"errors"
	"net"
)

// GetLocalIPv4 returns the first non-loopback IPv4 address of the host.
// If no suitable address is found, it returns an error.
func GetLocalIPv4() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue // skip non-IPNet or loopback addresses
		}
		ip := ipNet.IP.To4()
		if ip != nil {
			return ip.String(), nil // first valid IPv4 address
		}
	}

	return "", errors.New("no non-loopback IPv4 address found")
}
