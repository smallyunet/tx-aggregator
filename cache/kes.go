package cache

import (
	"fmt"
)

// formatChainKey generates a cache key for a specific chain with an address prefix
// Parameters:
//   - address: The address to use as a prefix
//   - chainID: The unique identifier of the blockchain
//
// Returns:
//   - string: The formatted cache key for the chain
func formatChainKey(address string, chainID int64) string {
	return fmt.Sprintf("%s-%d", address, chainID)
}

// formatTokenKey generates a cache key for a specific token on a specific chain with an address prefix
// Parameters:
//   - address: The address to use as a prefix
//   - chainID: The unique identifier of the blockchain
//   - tokenAddr: The address of the token contract
//
// Returns:
//   - string: The formatted cache key for the token
func formatTokenKey(address string, chainID int64, tokenAddr string) string {
	return fmt.Sprintf("%s-%d-%s", address, chainID, tokenAddr)
}

// formatTokenSetKey generates a cache key for the set of tokens on a specific chain with an address prefix
// Parameters:
//   - address: The address to use as a prefix
//   - chainID: The unique identifier of the blockchain
//
// Returns:
//   - string: The formatted cache key for the token set
func formatTokenSetKey(address string, chainID int64) string {
	return fmt.Sprintf("%s-%d-tokens", address, chainID)
}
