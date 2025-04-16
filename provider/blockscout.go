package provider

import (
	"golang.org/x/sync/errgroup"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// BlockscoutProvider implements the Provider interface for fetching transactions,
// token transfers, internal transactions, and logs from a Blockscout-compatible API.
type BlockscoutProvider struct {
	baseURL  string // Base URL for the Blockscout API, e.g., "https://api.blockscout.com/api/v2"
	chainID  int64  // Numeric chain ID
	chainKey string // Optional identifier for the chain, e.g., "bsc", "eth", etc.
	rpcURL   string // Optional RPC endpoint for fetching additional log data
}

// NewBlockscoutProvider creates a new BlockscoutProvider instance with the specified configuration.
// The baseURL will be trimmed of trailing slashes.
func NewBlockscoutProvider(baseURL string, chainID int64, chainKey, rpcURL string) *BlockscoutProvider {
	logger.Log.Info().
		Str("baseURL", baseURL).
		Str("rpcURL", rpcURL).
		Msg("Initializing new BlockscoutProvider")
	return &BlockscoutProvider{
		baseURL:  strings.TrimRight(baseURL, "/"),
		chainID:  chainID,
		chainKey: chainKey,
		rpcURL:   rpcURL,
	}
}

// GetTransactions concurrently fetches and combines normal transactions, token transfers,
// internal transactions, and logs for a given address. Logs are used to detect ERC20 approve calls.
func (t *BlockscoutProvider) GetTransactions(address string) (*model.TransactionResponse, error) {
	logger.Log.Info().
		Str("chain", t.chainKey).
		Str("address", address).
		Msg("Fetching transactions from Blockscout")

	var (
		normalTxs    []model.Transaction
		tokenTxs     []model.Transaction
		internalTxs  []model.Transaction
		logsResponse *model.BlockscoutLogResponse
		logsMap      = make(map[string][]model.BlockscoutLog) // txHash -> logs
	)

	// Concurrently fetch all types of transactions and logs
	g := new(errgroup.Group)

	// Fetch normal transactions
	g.Go(func() error {
		respData, err := t.fetchBlockscoutNormalTx(address)
		if err != nil {
			return err
		}
		normalTxs = t.transformBlockscoutNormalTx(respData, address, nil)
		return nil
	})

	// Fetch token transfers
	g.Go(func() error {
		respData, err := t.fetchBlockscoutTokenTransfers(address)
		if err != nil {
			return err
		}
		tokenTxs = t.transformBlockscoutTokenTransfers(respData, address)
		return nil
	})

	// Fetch internal transactions
	g.Go(func() error {
		respData, err := t.fetchBlockscoutInternalTx(address)
		if err != nil {
			return err
		}
		internalTxs = t.transformBlockscoutInternalTx(respData, address)
		return nil
	})

	// Fetch logs for approval detection
	g.Go(func() error {
		var err error
		logsResponse, err = t.fetchBlockscoutLogs(address)
		if err != nil {
			return err
		}
		logsMap = t.indexBlockscoutLogsByTxHash(logsResponse)
		return nil
	})

	// Wait for all concurrent fetches
	if err := g.Wait(); err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch some Blockscout data")
		return nil, err
	}

	// Optionally enrich normal transactions using RPC receipts if rpcURL is provided
	if len(normalTxs) > 0 && t.rpcURL != "" {
		uniqueBlocks := make(map[int64]bool)
		for _, tx := range normalTxs {
			uniqueBlocks[tx.Height] = true
		}

		rpcLogsMap, err := t.fetchLogsByBlockFromRPC(uniqueBlocks)
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to fetch logs from RPC")
		} else {
			normalTxs = t.transformBlockscoutNormalTxWithLogs(normalTxs, rpcLogsMap, address)
		}
	}

	// Re-process normal transactions with fetched logs
	if len(normalTxs) > 0 {
		normalTxs = t.transformBlockscoutNormalTxWithLogs(normalTxs, logsMap, address)
	}

	// Combine all transactions
	allTransactions := append(normalTxs, tokenTxs...)
	allTransactions = append(allTransactions, internalTxs...)

	logger.Log.Info().
		Int("normal_count", len(normalTxs)).
		Int("token_count", len(tokenTxs)).
		Int("internal_count", len(internalTxs)).
		Int("total_transactions", len(allTransactions)).
		Str("chain", t.chainKey).
		Str("address", address).
		Msg("Successfully fetched and merged all Blockscout transactions")

	return &model.TransactionResponse{
		Result: struct {
			Transactions []model.Transaction `json:"transactions"`
		}{
			Transactions: allTransactions,
		},
		Id: int(t.chainID),
	}, nil
}
