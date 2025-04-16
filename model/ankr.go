package model

// AnkrTransactionRequest represents the request structure for Ankr API transaction queries
type AnkrTransactionRequest struct {
	JSONRPC string                 `json:"jsonrpc"` // JSON-RPC version
	Method  string                 `json:"method"`  // API method name
	Params  map[string]interface{} `json:"params"`  // Method parameters
	ID      int                    `json:"id"`      // Request identifier
}

// AnkrTransactionResponse represents the response structure for Ankr API transaction queries
type AnkrTransactionResponse struct {
	JSONRPC string `json:"jsonrpc"` // JSON-RPC version
	ID      int    `json:"id"`      // Request identifier
	Result  struct {
		Transactions []AnkrTransaction `json:"transactions"` // List of transactions
	} `json:"result"`
}

// AnkrTokenTransferResponse represents the response structure for Ankr API token transfer queries
type AnkrTokenTransferResponse struct {
	JSONRPC string `json:"jsonrpc"` // JSON-RPC version
	ID      int    `json:"id"`      // Request identifier
	Result  struct {
		NextPageToken string          `json:"nextPageToken"` // Token for pagination
		Transfers     []TokenTransfer `json:"transfers"`     // List of token transfers
	} `json:"result"`
}

// AnkrLogEntry represents a blockchain log entry from a transaction
type AnkrLogEntry struct {
	Blockchain       string   `json:"blockchain"`       // Blockchain network identifier
	Address          string   `json:"address"`          // Contract address that emitted the log
	Topics           []string `json:"topics"`           // Log topics/events
	Data             string   `json:"data"`             // Log data in hex format
	BlockNumber      string   `json:"blockNumber"`      // Block number where the log was created
	TransactionHash  string   `json:"transactionHash"`  // Hash of the transaction
	TransactionIndex string   `json:"transactionIndex"` // Index of the transaction in the block
	BlockHash        string   `json:"blockHash"`        // Hash of the block
	LogIndex         string   `json:"logIndex"`         // Index of the log in the transaction
	Removed          bool     `json:"removed"`          // Whether the log was removed due to chain reorganization
	Timestamp        string   `json:"timestamp"`        // Timestamp when the log was created
}

// AnkrTransaction represents a blockchain transaction with all its details
type AnkrTransaction struct {
	BlockHash         string         `json:"blockHash"`         // Hash of the block containing this transaction
	BlockNumber       string         `json:"blockNumber"`       // Number of the block containing this transaction
	Blockchain        string         `json:"blockchain"`        // Blockchain network identifier
	CumulativeGasUsed string         `json:"cumulativeGasUsed"` // Total gas used in the block up to this transaction
	From              string         `json:"from"`              // Sender address
	Gas               string         `json:"gas"`               // Gas limit for the transaction
	GasPrice          string         `json:"gasPrice"`          // Gas price in wei
	GasUsed           string         `json:"gasUsed"`           // Gas used by the transaction
	Hash              string         `json:"hash"`              // Transaction hash
	Input             string         `json:"input"`             // Transaction input data
	Nonce             string         `json:"nonce"`             // Transaction nonce
	R                 string         `json:"r"`                 // ECDSA signature r value
	S                 string         `json:"s"`                 // ECDSA signature s value
	Status            string         `json:"status"`            // Transaction status (0: failed, 1: success)
	Timestamp         string         `json:"timestamp"`         // Transaction timestamp
	To                string         `json:"to"`                // Recipient address
	TransactionIndex  string         `json:"transactionIndex"`  // Index of the transaction in the block
	Type              string         `json:"type"`              // Transaction type
	V                 string         `json:"v"`                 // ECDSA signature v value
	Value             string         `json:"value"`             // Transaction value in wei
	Logs              []AnkrLogEntry `json:"logs"`              // Transaction event logs
}

// TokenTransfer represents a token transfer event
type TokenTransfer struct {
	FromAddress     string `json:"fromAddress"`     // Sender address
	ToAddress       string `json:"toAddress"`       // Recipient address
	ContractAddress string `json:"contractAddress"` // Token contract address
	Value           string `json:"value"`           // Transfer amount in token units
	ValueRawInteger string `json:"valueRawInteger"` // Transfer amount in raw integer format
	TokenName       string `json:"tokenName"`       // Name of the token
	TokenSymbol     string `json:"tokenSymbol"`     // Symbol of the token
	TokenDecimals   int    `json:"tokenDecimals"`   // Number of decimal places for the token
	TransactionHash string `json:"transactionHash"` // Hash of the transaction
	BlockHeight     int    `json:"blockHeight"`     // Block height of the transfer
	Timestamp       int64  `json:"timestamp"`       // Timestamp of the transfer
	Blockchain      string `json:"blockchain"`      // Blockchain network identifier
	Thumbnail       string `json:"thumbnail"`       // URL to token thumbnail/logo image
}
