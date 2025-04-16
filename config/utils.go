package config

// ChainIDByName returns the chain ID for a given chain name.
// If the chain name is not found in the configuration, it returns 0.
// This function is used to map chain names to their corresponding IDs
// as defined in the application configuration.
func ChainIDByName(name string) int64 {
	if id, ok := AppConfig.ChainIDs[name]; ok {
		return id
	}
	return 0
}
