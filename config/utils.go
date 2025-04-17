package config

import (
	"fmt"
	"strings"
)

// ChainIDByName returns the chain ID for a given chain name (case-insensitive).
// If the chain name is not found, it returns an error.
func ChainIDByName(name string) (int64, error) {
	upperName := strings.ToUpper(name)
	for key, id := range AppConfig.ChainIDs {
		if strings.ToUpper(key) == upperName {
			return id, nil
		}
	}
	return 0, fmt.Errorf("unknown chain name: %s", name)
}

// ChainNameByID returns the UPPERCASE chain name for a given chain ID.
// If not found, it returns an error.
func ChainNameByID(id int64) (string, error) {
	for name, chainID := range AppConfig.ChainIDs {
		if chainID == id {
			return strings.ToUpper(name), nil
		}
	}
	return "", fmt.Errorf("unknown chain ID: %d", id)
}
