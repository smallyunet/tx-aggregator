package config

import "log"

// ChainIDByName returns the chain ID for a given chain name.
// If the chain name is not found in the configuration, it returns 0.
// This function is used to map chain names to their corresponding IDs
// as defined in the application configuration.
func ChainIDByName(name string) int64 {
	log.Printf("Looking up chain ID for chain name: %s", name)
	if id, ok := AppConfig.ChainIDs[name]; ok {
		log.Printf("Found chain ID %d for chain name: %s", id, name)
		return id
	}
	log.Printf("Chain name not found in configuration: %s", name)
	return 0
}
