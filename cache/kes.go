package cache

import (
	"fmt"
	"strings"
	"tx-aggregator/model"
)

// formatChainKey generates a cache key for a specific chain with an address prefix.
// The chain name is converted to lowercase to ensure case-insensitive consistency.
func formatChainKey(address, chainName string) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(address), strings.ToLower(chainName))
}

// formatNativeKey generates a cache key for the native token on a specific chain with an address prefix.
// The chain name is converted to lowercase to ensure case-insensitive consistency.
func formatNativeKey(address, chainName string) string {
	return fmt.Sprintf("%s-%s-%s", strings.ToLower(address), strings.ToLower(chainName), model.NativeTokenName)
}

// formatTokenKey generates a cache key for a specific token on a specific chain with an address prefix.
// Both address and chainName are normalized to lowercase.
func formatTokenKey(address, chainName, tokenAddr string) string {
	return fmt.Sprintf("%s-%s-%s", strings.ToLower(address), strings.ToLower(chainName), strings.ToLower(tokenAddr))
}

// formatTokenSetKey generates a cache key for the set of tokens on a specific chain with an address prefix.
// The chain name is normalized to lowercase.
func formatTokenSetKey(address, chainName string) string {
	return fmt.Sprintf("%s-%s-tokens", strings.ToLower(address), strings.ToLower(chainName))
}
