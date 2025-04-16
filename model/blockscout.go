package model

// ===== NORMAL TRANSACTIONS =====

// BlockscoutTransactionResponse represents the response from
// /addresses/{address}/transactions endpoint, listing normal transactions.
type BlockscoutTransactionResponse struct {
	Items []BlockscoutTransaction `json:"items"`
}

// BlockscoutTransaction represents a single normal transaction in the Tantin response.
type BlockscoutTransaction struct {
	Hash             string                     `json:"hash"`              // Transaction hash
	BlockHash        string                     `json:"block_hash"`        // Block hash
	BlockNumber      int64                      `json:"block_number"`      // Block number as integer
	Value            string                     `json:"value"`             // Value transferred in Wei
	GasUsed          string                     `json:"gas_used"`          // Gas used for the transaction
	GasLimit         string                     `json:"gas_limit"`         // Gas limit set by the sender
	GasPrice         string                     `json:"gas_price"`         // Gas price used
	Timestamp        string                     `json:"timestamp"`         // ISO timestamp, e.g. "2025-04-16T06:45:02.000000Z"
	Nonce            int64                      `json:"nonce"`             // Nonce of the transaction
	Status           string                     `json:"status"`            // "ok" for success, others for failure
	Method           string                     `json:"method"`            // Method name if known (contract calls)
	From             BlockscoutAddressContainer `json:"from"`              // Sender address container
	To               BlockscoutAddressContainer `json:"to"`                // Recipient address container
	TransactionTypes []string                   `json:"transaction_types"` // Types of transaction, e.g. ["contract_call", "token_transfer"]
}

// BlockscoutAddressContainer represents a simple address object with only hash (used in normal txs).
type BlockscoutAddressContainer struct {
	Hash string `json:"hash"` // Address hash
}

// ===== TOKEN TRANSFERS =====

// BlockscoutTokenTransferResponse represents the response from
// /addresses/{address}/token-transfers endpoint.
type BlockscoutTokenTransferResponse struct {
	Items []BlockscoutTokenTransfer `json:"items"`
}

// BlockscoutTokenTransfer represents a single token transfer event.
type BlockscoutTokenTransfer struct {
	BlockHash       string                     `json:"block_hash"`       // Block hash
	BlockNumber     int64                      `json:"block_number"`     // Block number
	From            BlockscoutAddressContainer `json:"from"`             // Sender address
	To              BlockscoutAddressContainer `json:"to"`               // Recipient address
	Timestamp       string                     `json:"timestamp"`        // ISO timestamp
	TransactionHash string                     `json:"transaction_hash"` // Transaction hash
	Token           BlockscoutTokenInfo        `json:"token"`            // Token metadata
	Total           BlockscoutTokenAmount      `json:"total"`            // Transfer amount
	Type            string                     `json:"type"`             // e.g. "token_transfer", "token_minting"
}

// BlockscoutTokenInfo holds metadata about a token involved in a transfer.
type BlockscoutTokenInfo struct {
	Address  string `json:"address"`  // Token contract address
	Decimals string `json:"decimals"` // Number of decimal places
	IconURL  string `json:"icon_url"` // URL to the token's icon
	Name     string `json:"name"`     // Human-readable token name
	Symbol   string `json:"symbol"`   // Token symbol, e.g. "USDT"
}

// BlockscoutTokenAmount represents the transferred token amount.
type BlockscoutTokenAmount struct {
	Decimals string `json:"decimals"` // Number of decimals
	Value    string `json:"value"`    // Token amount in smallest unit
}

// ===== INTERNAL TRANSACTIONS =====

// BlockscoutInternalTxResponse represents the response from
// /addresses/{address}/internal-transactions endpoint.
type BlockscoutInternalTxResponse struct {
	Items []BlockscoutInternalTx `json:"items"`
}

// BlockscoutInternalTx represents a single internal transaction.
type BlockscoutInternalTx struct {
	BlockNumber     int64                     `json:"block_number"`     // Block number
	CreatedContract *BlockscoutAddressDetails `json:"created_contract"` // Contract created, if any
	Error           string                    `json:"error"`            // Error message if failed
	From            *BlockscoutAddressDetails `json:"from"`             // Caller address
	To              *BlockscoutAddressDetails `json:"to"`               // Callee address
	GasLimit        string                    `json:"gas_limit"`        // Gas limit
	Index           int64                     `json:"index"`            // Index in internal tx list
	Success         bool                      `json:"success"`          // Whether the call succeeded
	Timestamp       string                    `json:"timestamp"`        // ISO timestamp
	TransactionHash string                    `json:"transaction_hash"` // Parent transaction hash
	Type            string                    `json:"type"`             // Call type, e.g. "call", "create"
	Value           string                    `json:"value"`            // Value transferred in Wei
}

// BlockscoutAddressDetails provides detailed info about an address in internal tx/logs.
type BlockscoutAddressDetails struct {
	Hash               string      `json:"hash"`                // Address hash
	ImplementationName string      `json:"implementation_name"` // Name of the contract implementation
	Name               string      `json:"name"`                // Contract or user-defined name
	EnsDomainName      string      `json:"ens_domain_name"`     // ENS name if available
	Metadata           interface{} `json:"metadata"`            // Optional metadata
	IsContract         bool        `json:"is_contract"`         // True if it's a smart contract
	IsVerified         bool        `json:"is_verified"`         // True if contract source is verified
	// Note: other fields like private/public tags may be present
}

// ===== LOGS =====

// BlockscoutLogResponse represents the response from /addresses/{address}/logs endpoint.
type BlockscoutLogResponse struct {
	Items []BlockscoutLog `json:"items"`
}

// BlockscoutLog represents an individual log/event emitted by a smart contract.
type BlockscoutLog struct {
	Address         BlockscoutAddressDetails `json:"address"`          // Log origin address
	BlockHash       string                   `json:"block_hash"`       // Block hash
	BlockNumber     int64                    `json:"block_number"`     // Block number
	Data            string                   `json:"data"`             // Raw log data
	Decoded         *BlockscoutLogDecoded    `json:"decoded"`          // Optional decoded data
	Index           int64                    `json:"index"`            // Log index in block
	SmartContract   BlockscoutAddressDetails `json:"smart_contract"`   // Contract that emitted the log
	Topics          []string                 `json:"topics"`           // Indexed event topics
	TransactionHash string                   `json:"transaction_hash"` // Parent transaction hash
}

// BlockscoutLogDecoded holds the decoded log data if ABI was available.
type BlockscoutLogDecoded struct {
	MethodCall string `json:"method_call"` // Name of the method/event
	MethodID   string `json:"method_id"`   // Method ID (first 4 bytes of selector)
	Parameters []struct {
		Name    string `json:"name"`    // Parameter name
		Type    string `json:"type"`    // Solidity type
		Value   string `json:"value"`   // Value as string
		Indexed bool   `json:"indexed"` // Whether the parameter is indexed
	} `json:"parameters"`
}

// ===== RPC RECEIPT STRUCTURES =====

// RpcReceiptLog represents an Ethereum log as returned by eth_getTransactionReceipt.
type RpcReceiptLog struct {
	Address          string   `json:"address"`          // Log origin address
	Topics           []string `json:"topics"`           // List of indexed topics
	Data             string   `json:"data"`             // Non-indexed data
	BlockNumber      string   `json:"blockNumber"`      // Block number (hex string)
	TransactionHash  string   `json:"transactionHash"`  // Tx hash
	TransactionIndex string   `json:"transactionIndex"` // Index of tx in block
	BlockHash        string   `json:"blockHash"`        // Block hash
	LogIndex         string   `json:"logIndex"`         // Index of log in tx
	Removed          bool     `json:"removed"`          // True if log was removed due to reorg
}

// RpcReceipt represents the full transaction receipt structure from JSON-RPC.
type RpcReceipt struct {
	BlockHash         string          `json:"blockHash"`         // Block hash
	BlockNumber       string          `json:"blockNumber"`       // Block number
	ContractAddress   string          `json:"contractAddress"`   // New contract address (if created)
	CumulativeGasUsed string          `json:"cumulativeGasUsed"` // Total gas used in block up to this tx
	EffectiveGasPrice string          `json:"effectiveGasPrice"` // Actual gas price paid
	From              string          `json:"from"`              // Sender address
	GasUsed           string          `json:"gasUsed"`           // Gas used by this transaction
	Logs              []RpcReceiptLog `json:"logs"`              // Event logs emitted
	LogsBloom         string          `json:"logsBloom"`         // Bloom filter for quick lookup
	Status            string          `json:"status"`            // "0x1" for success, "0x0" for failure
	To                string          `json:"to"`                // Recipient address
	TransactionHash   string          `json:"transactionHash"`   // Transaction hash
	TransactionIndex  string          `json:"transactionIndex"`  // Index in block
	Type              string          `json:"type"`              // Transaction type
}

// RpcReceiptResponse represents a batched response from JSON-RPC call for receipts.
type RpcReceiptResponse struct {
	ID      int          `json:"id"`      // Request ID
	JSONRPC string       `json:"jsonrpc"` // JSON-RPC version
	Result  []RpcReceipt `json:"result"`  // Array of receipt results
}
