package provider

import (
	"golang.org/x/sync/errgroup"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// BlockscoutProvider implements the Provider interface for fetching transactions,
// token transfers, internal transactions, and logs from the Tantin (Blockscout-like) API.
type BlockscoutProvider struct {
	baseURL  string // Base URL for the Tantin API, e.g. "https://api.tantin.com/api/v2"
	chainID  int64  // Numeric chain ID
	chainKey string // Optional identifier for the chain, e.g. "bsc", "eth", etc.
	rpcURL   string
}

// NewBlockscoutProvider creates a new BlockscoutProvider with the specified baseURL, chainID, and chainKey.
// baseURL is trimmed to remove any trailing slashes.
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

// GetTransactions fetches and combines normal transactions, token transfers, internal transactions,
// and logs for a given address. Logs are used to detect whether a transaction is an "approve" transaction,
// which is then shown in the normal transaction list (approveShow).
func (t *BlockscoutProvider) GetTransactions(address string) (*model.TransactionResponse, error) {
	logger.Log.Info().
		Str("chain", t.chainKey).
		Str("address", address).
		Msg("Fetching transactions from Tantin")

	var (
		normalTxs    []model.Transaction
		tokenTxs     []model.Transaction
		internalTxs  []model.Transaction
		logsResponse *model.BlockscoutLogResponse

		// We will store the logs in a map keyed by transaction hash
		logsMap = make(map[string][]model.BlockscoutLog)
	)

	// Use errgroup for concurrent requests
	g := new(errgroup.Group)

	// 1. Fetch & transform normal transactions
	g.Go(func() error {
		respData, err := t.fetchBlockscoutNormalTx(address)
		if err != nil {
			return err
		}
		// We'll transform them later, after we fetch logs (to determine if Approve).
		// Just store them in a local variable for now.
		normalTxs = t.transformBlockscoutNormalTx(respData, address, nil)
		return nil
	})

	// 2. Fetch & transform token transfers
	g.Go(func() error {
		respData, err := t.fetchTantinTokenTransfers(address)
		if err != nil {
			return err
		}
		tokenTxs = t.transformBlockscoutTokenTransfers(respData, address)
		return nil
	})

	// 3. Fetch & transform internal transactions
	g.Go(func() error {
		respData, err := t.fetchBlockscoutInternalTx(address)
		if err != nil {
			return err
		}
		internalTxs = t.transformBlockscoutInternalTx(respData, address)
		return nil
	})

	// 4. Fetch logs (to detect "approve" info). We'll store them for usage in normalTxs transformation.
	// TODO: Tantin explorer does not support logs fetching by address
	// https://api.tantin.com/api/v2/addresses/0x472e93D8Ba72345cfCE0800eE24A8f69705a814D/internal-transactions
	g.Go(func() error {
		var err error
		logsResponse, err = t.fetchBlockscoutLogs(address)
		if err != nil {
			return err
		}
		logsMap = t.indexBlockscoutLogsByTxHash(logsResponse)
		return nil
	})

	// Wait for all concurrent requests to finish
	if err := g.Wait(); err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch some Tantin data")
		return nil, err
	}

	if len(normalTxs) > 0 && t.rpcURL != "" {
		// 1. Collect all block numbers
		uniqueBlocks := make(map[int64]bool)
		for _, tx := range normalTxs {
			uniqueBlocks[tx.Height] = true
		}

		// 2. Call RPC to fetch receipts for these blocks
		rpcLogsMap, err := t.fetchLogsByBlockFromRPC(uniqueBlocks)
		if err != nil {
			logger.Log.Error().Err(err).Msg("Failed to fetch logs from RPC")
		} else {
			// 3. Reuse the existing transform method to reprocess normalTxs
			//    to detect approve/transfer types.
			//    Note that we directly reuse transformBlockscoutNormalTxWithLogs,
			//    as long as we convert rpcLogsMap to []BlockscoutLog, it can be used.
			normalTxs = t.transformBlockscoutNormalTxWithLogs(normalTxs, rpcLogsMap, address)
		}
	}

	// Re-transform normal transactions now that we have logsMap
	if len(normalTxs) > 0 {
		normalTxs = t.transformBlockscoutNormalTxWithLogs(normalTxs, logsMap, address)
	}

	// Merge the final results
	allTransactions := append(normalTxs, tokenTxs...)
	allTransactions = append(allTransactions, internalTxs...)

	logger.Log.Info().
		Int("normal_count", len(normalTxs)).
		Int("token_count", len(tokenTxs)).
		Int("internal_count", len(internalTxs)).
		Int("total_transactions", len(allTransactions)).
		Str("chain", t.chainKey).
		Str("address", address).
		Msg("Successfully fetched and merged all Tantin transactions")

	return &model.TransactionResponse{
		Result: struct {
			Transactions []model.Transaction `json:"transactions"`
		}{
			Transactions: allTransactions,
		},
		Id: int(t.chainID),
	}, nil
}
