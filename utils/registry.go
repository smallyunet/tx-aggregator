package utils

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

// ResolveAnkrBlockchains converts user-supplied chain names (e.g. "ETH", "BSC")
// to the lowercase identifiers expected by the Ankr API.
//
// Rules
//   - If paramNames is empty, fall back to AppConfig.Ankr.RequestBlockchains.
//   - Otherwise:
//     1. Convert each name → chain ID via ChainIDByName.
//     2. Reverse-lookup that ID in AppConfig.Ankr.ChainIDs to get the
//     corresponding Ankr name (eth, bsc, polygon, base …).
//   - Any unknown name or unsupported chain triggers an error.
//
// Returned slice contains unique, lowercase Ankr names.
func ResolveAnkrBlockchains(paramNames []string) ([]string, error) {
	// Return defaults when the caller did not specify chainNames.
	if len(paramNames) == 0 {
		return config.AppConfig.Ankr.RequestBlockchains, nil
	}

	// Build reverse index: chainID → ankrName.
	idToAnkr := make(map[int64]string, len(config.AppConfig.Ankr.ChainIDs))
	for ankrName, id := range config.AppConfig.Ankr.ChainIDs {
		idToAnkr[id] = ankrName
	}

	var blockchains []string
	seen := make(map[string]struct{}) // de-duplication

	for _, raw := range paramNames {
		chainID, err := ChainIDByName(raw)
		if err != nil {
			return nil, err // unknown chain name
		}
		ankrName, ok := idToAnkr[chainID]
		if !ok {
			return nil, fmt.Errorf("chain %s (id=%d) not supported by Ankr provider", raw, chainID)
		}
		ankrName = strings.ToLower(ankrName)
		if _, dup := seen[ankrName]; !dup {
			blockchains = append(blockchains, ankrName)
			seen[ankrName] = struct{}{}
		}
	}

	return blockchains, nil
}
