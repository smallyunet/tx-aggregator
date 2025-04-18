package chainmeta

import (
	"fmt"
	"strings"
	"tx-aggregator/config"
)

// ChainIDByName returns the chain ID for a given chain name (case-insensitive).
// If the chain name is not found, it returns an error.
func ChainIDByName(name string) (int64, error) {
	upperName := strings.ToUpper(name)
	for key, id := range config.AppConfig.ChainNames {
		if strings.ToUpper(key) == upperName {
			return id, nil
		}
	}
	return 0, fmt.Errorf("unknown chain name: %s", name)
}

// ChainNameByID returns the UPPERCASE chain name for a given chain ID.
// If not found, it returns an error.
func ChainNameByID(id int64) (string, error) {
	for name, chainID := range config.AppConfig.ChainNames {
		if chainID == id {
			return strings.ToUpper(name), nil
		}
	}
	return "", fmt.Errorf("unknown chain ID: %d", id)
}

// AnkrChainIDByName returns the chain ID for a given Ankr chain name (case-insensitive).
// If the chain name is not found, it returns an error.
func AnkrChainIDByName(name string) (int64, error) {
	upperName := strings.ToUpper(name)
	for key, id := range config.AppConfig.Ankr.ChainIDs {
		if strings.ToUpper(key) == upperName {
			return id, nil
		}
	}
	return 0, fmt.Errorf("unknown chain name: %s", name)
}

// AnkrChainNameByID returns the UPPERCASE chain name for a given Ankr chain ID.
// If not found, it returns an error.
func AnkrChainNameByID(id int64) (string, error) {
	for name, chainID := range config.AppConfig.Ankr.ChainIDs {
		if chainID == id {
			return strings.ToUpper(name), nil
		}
	}
	return "", fmt.Errorf("unknown chain ID: %d", id)
}
