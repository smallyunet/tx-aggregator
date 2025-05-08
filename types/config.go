package types

// Config represents the application configuration structure
type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Providers    ProvidersConfig    `mapstructure:"providers"`
	Ankr         AnkrConfig         `mapstructure:"ankr"`
	Blockscout   []BlockscoutConfig `mapstructure:"blockscout"`
	Log          LogConfig          `mapstructure:"log"`
	Response     ResponseConfig     `mapstructure:"response"`
	ChainNames   map[string]int64   `mapstructure:"chain_names"`
	NativeTokens map[string]string  `mapstructure:"native_tokens"`
	Blockscan    []BlockscanConfig  `mapstructure:"blockscan"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port int `mapstructure:"port"` // Use int to match YAML
}

// RedisConfig holds Redis connection details.
type RedisConfig struct {
	Addrs      []string `mapstructure:"addrs"`
	Password   string   `mapstructure:"password"`
	TTLSeconds int      `mapstructure:"ttl"`
}

// ProvidersConfig holds provider-level settings.
type ProvidersConfig struct {
	RequestTimeout int64             `mapstructure:"request_timeout"`
	ChainProviders map[string]string `mapstructure:"chain_providers"`
}

// AnkrConfig holds Ankr provider settings.
type AnkrConfig struct {
	APIKey          string           `mapstructure:"api_key"`
	URL             string           `mapstructure:"url"`
	RequestPageSize int              `mapstructure:"request_page_size"`
	ChainIDs        map[string]int64 `mapstructure:"chain_ids"`
}

// BlockscoutConfig represents a single Blockscout instance configuration.
type BlockscoutConfig struct {
	URL               string `mapstructure:"url"`
	ChainName         string `mapstructure:"chain_name"`
	RequestPageSize   int64  `mapstructure:"request_page_size"`
	RPCURL            string `mapstructure:"rpc_url"`
	RPCRequestTimeout int64  `mapstructure:"rpc_request_timeout"`
}

// LogConfig holds logging level.
type LogConfig struct {
	Level         int8   `mapstructure:"level"`
	Path          string `mapstructure:"path"`
	ConsoleFormat string `mapstructure:"console_format"`
	FileFormat    string `mapstructure:"file_format"`
}

// ResponseConfig limits response size.
type ResponseConfig struct {
	Max int `mapstructure:"max"`
}

// BlockscanConfig holds per-chain settings for BscScan / Etherscan style APIs.
type BlockscanConfig struct {
	URL             string `mapstructure:"url"`               // e.g. https://api-testnet.bscscan.com/api
	APIKey          string `mapstructure:"api_key"`           // personal API key
	ChainName       string `mapstructure:"chain_name"`        // BSC, ETH, etc. – used in YAML mapping
	RequestPageSize int    `mapstructure:"request_page_size"` // Max items per page (100 is typical)
}
