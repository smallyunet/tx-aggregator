package utils

import (
	"fmt"
	"slices"
	"strings"
	"tx-aggregator/config"
)

// ChainIDByName returns the chain ID for a given chain name (case-insensitive).
// If the chain name is not found, it returns an error.
func ChainIDByName(name string) (int64, error) {
	upperName := strings.ToUpper(name)
	for key, id := range config.Current().ChainNames {
		if strings.ToUpper(key) == upperName {
			return id, nil
		}
	}
	return 0, fmt.Errorf("unknown chain name: %s", name)
}

// ChainNameByID returns the UPPERCASE chain name for a given chain ID.
// If not found, it returns an error.
func ChainNameByID(id int64) (string, error) {
	for name, chainID := range config.Current().ChainNames {
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
	for key, id := range config.Current().Ankr.ChainIDs {
		if strings.ToUpper(key) == upperName {
			return id, nil
		}
	}
	return 0, fmt.Errorf("unknown chain name: %s", name)
}

// AnkrChainNameByID returns the UPPERCASE chain name for a given Ankr chain ID.
// If not found, it returns an error.
func AnkrChainNameByID(id int64) (string, error) {
	for name, chainID := range config.Current().Ankr.ChainIDs {
		if chainID == id {
			return strings.ToUpper(name), nil
		}
	}
	return "", fmt.Errorf("unknown chain ID: %d", id)
}

// ResolveAnkrBlockchains converts UI-supplied chain names into the lowercase
// identifiers expected by the Ankr Multichain API.
//
// Behaviour
// • Empty input → return all chains supported by Ankr (derived from
// cfg.Ankr.ChainIDs).
// • Unknown chain names, or chain names not supported by Ankr, are skipped.
// • If filtering leaves the slice empty, fall back to the full supported set.
//
// Example:
//
// paramNames := []string{"ETH", "FOO", "BSC"}
// // → []string{"eth", "bsc"}
func ResolveAnkrBlockchains(paramNames []string) ([]string, error) {
	cfg := config.Current()
	// helper: return all supported chains (deduplicated, lowercase, sorted)
	allSupported := func() []string {
		out := make([]string, 0, len(cfg.Ankr.ChainIDs))
		for name := range cfg.Ankr.ChainIDs {
			out = append(out, strings.ToLower(name))
		}
		slices.Sort(out) // optional: ensures deterministic order
		return out
	}

	// 1. No names provided → return full supported list.
	if len(paramNames) == 0 {
		return allSupported(), nil
	}

	// 2. Build reverse index: chainID → ankrName.
	idToAnkr := make(map[int64]string, len(cfg.Ankr.ChainIDs))
	for ankrName, id := range cfg.Ankr.ChainIDs {
		idToAnkr[id] = ankrName
	}

	var blockchains []string
	seen := make(map[string]struct{}) // deduplication

	// 3. Translate & filter.
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

	// 4. Everything filtered out → return full supported list.
	if len(blockchains) == 0 {
		return allSupported(), nil
	}

	// Optional: sort result for deterministic output
	slices.Sort(blockchains)
	return blockchains, nil
}

// NativeTokenByChainID returns the native token name for a given chain ID.
// If the chain ID is not found in the configuration, it returns an error.
func NativeTokenByChainID(id int64) (string, error) {
	idStr := fmt.Sprintf("%d", id)
	token, ok := config.Current().NativeTokens[idStr]
	if !ok {
		return "", fmt.Errorf("native token not found for chain ID: %d", id)
	}
	return token, nil
}
