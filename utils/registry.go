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

// ResolveAnkrBlockchains converts user–supplied chain names to the lowercase
// identifiers required by the Ankr API.
//
// Behaviour change:
//   • Any unknown or unsupported chain name is *ignored* instead of causing
//     an error.
//   • If, after filtering, the list is empty, the function falls back to
//     AppConfig.Ankr.RequestBlockchains so that the caller still gets a
//     valid slice.
//
// Example
//   paramNames = []string{"ETH", "FOO", "BSC"}
//   → returns []string{"eth", "bsc"}
func ResolveAnkrBlockchains(paramNames []string) ([]string, error) {
	// 1. No names supplied → use defaults immediately.
	if len(paramNames) == 0 {
		return config.AppConfig.Ankr.RequestBlockchains, nil
	}

	// 2. Build reverse index: chain-ID → ankrName.
	idToAnkr := make(map[int64]string, len(config.AppConfig.Ankr.ChainIDs))
	for ankrName, id := range config.AppConfig.Ankr.ChainIDs {
		idToAnkr[id] = ankrName
	}

	var blockchains []string
	seen := make(map[string]struct{}) // de-duplication

	// 3. Translate / filter.
	for _, raw := range paramNames {
		chainID, err := ChainIDByName(raw) // unknown UI name → skip
		if err != nil {
			continue
		}
		ankrName, ok := idToAnkr[chainID] // supported by Ankr?
		if !ok {
			continue
		}
		ankrName = strings.ToLower(ankrName)
		if _, dup := seen[ankrName]; !dup {
			blockchains = append(blockchains, ankrName)
			seen[ankrName] = struct{}{}
		}
	}

	// 4. If everything was filtered out, revert to defaults.
	if len(blockchains) == 0 {
		return config.AppConfig.Ankr.RequestBlockchains, nil
	}
	return blockchains, nil
}
