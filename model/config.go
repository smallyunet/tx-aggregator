package model

// Config represents the application configuration structure
type Config struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`

	Redis struct {
		Addrs    []string `mapstructure:"addrs"`
		Password string   `mapstructure:"password"`
	} `mapstructure:"redis"`

	Ankr struct {
		APIKey             string           `mapstructure:"api_key"`
		URL                string           `mapstructure:"url"`
		RequestBlockchains []string         `mapstructure:"request_blockchains"`
		RequestPageSize    int              `mapstructure:"request_page_size"`
		ChainIDs           map[string]int64 `mapstructure:"chain_ids"`
	} `mapstructure:"ankr"`

	Blockscout []BlockscoutConfig `mapstructure:"blockscout"`

	Cache struct {
		TTLSeconds int `mapstructure:"ttl"`
	} `mapstructure:"cache"`

	Log struct {
		Level int8 `mapstructure:"level"`
	} `mapstructure:"log"`

	Response struct {
		Max int `mapstructure:"max"`
	} `mapstructure:"response"`

	ChainNames map[string]int64 `mapstructure:"chain_names"`
}

// BlockscoutConfig represents a single Blockscout instance configuration.
type BlockscoutConfig struct {
	URL             string `mapstructure:"url"`
	ChainName       string `mapstructure:"chain_name"`
	RPCURL          string `mapstructure:"rpc_url"`
	RequestPageSize int64  `mapstructure:"request_page_size"`
}
