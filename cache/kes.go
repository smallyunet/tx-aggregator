package cache

import (
	"fmt"
	"log"
)

// formatChainKey generates a cache key for a specific chain
// Parameters:
//   - chainID: The unique identifier of the blockchain
//
// Returns:
//   - string: The formatted cache key for the chain
func formatChainKey(chainID int64) string {
	log.Printf("Formatting chain key for chainID: %d", chainID)
	return fmt.Sprintf("%d", chainID)
}

// formatTokenKey generates a cache key for a specific token on a specific chain
// Parameters:
//   - chainID: The unique identifier of the blockchain
//   - tokenAddr: The address of the token contract
//
// Returns:
//   - string: The formatted cache key for the token
func formatTokenKey(chainID int64, tokenAddr string) string {
	log.Printf("Formatting token key for chainID: %d, tokenAddr: %s", chainID, tokenAddr)
	return fmt.Sprintf("%d-%s", chainID, tokenAddr)
}

// formatTokenSetKey generates a cache key for the set of tokens on a specific chain
// Parameters:
//   - chainID: The unique identifier of the blockchain
//
// Returns:
//   - string: The formatted cache key for the token set
func formatTokenSetKey(chainID int64) string {
	log.Printf("Formatting token set key for chainID: %d", chainID)
	return fmt.Sprintf("%d-tokens", chainID)
}
