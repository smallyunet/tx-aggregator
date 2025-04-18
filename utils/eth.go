package utils

import (
	"encoding/hex"
	"strings"
)

// IsValidEthereumAddress checks if the given address is a valid Ethereum address.
func IsValidEthereumAddress(addr string) bool {
	// Should start with '0x', be 42 characters long, and be valid hex
	if len(addr) != 42 || !strings.HasPrefix(addr, "0x") {
		return false
	}
	_, err := hex.DecodeString(addr[2:])
	return err == nil
}
