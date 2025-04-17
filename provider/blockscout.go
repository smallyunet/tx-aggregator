// File: provider/blockscout_provider.go
// Package provider implements data sources for the transaction‑aggregator service.
// The BlockscoutProvider fetches transactions, token transfers, internal
// transactions, and logs from a Blockscout‑compatible REST API, and (optionally)
// extra logs from an RPC endpoint. All comments are in English as requested.

package provider

import (
	"tx-aggregator/logger"
	"tx-aggregator/model"

	"golang.org/x/sync/errgroup"
)

// BlockscoutProvider implements the Provider interface for fetching transaction
// data from a Blockscout‑compatible API.
type BlockscoutProvider struct {
	chainID int64 // Numeric chain ID
	config  model.BlockscoutConfig
}

// NewBlockscoutProvider returns a new BlockscoutProvider.
// Trailing slashes are trimmed from baseURL for consistency.
func NewBlockscoutProvider(chainID int64, config model.BlockscoutConfig) *BlockscoutProvider {
	logger.Log.Info().
		Msg("Initializing BlockscoutProvider")

	return &BlockscoutProvider{
		chainID: chainID,
		config:  config,
	}
}

// GetTransactions concurrently fetches all relevant data for a single address
// and returns a unified TransactionResponse.
func (p *BlockscoutProvider) GetTransactions(address string) (*model.TransactionResponse, error) {
	logger.Log.Info().
		Str("chain", p.config.ChainName).
		Str("address", address).
		Msg("Fetching transactions from Blockscout")

	var (
		normalTxs   []model.Transaction
		tokenTxs    []model.Transaction
		internalTxs []model.Transaction

		// allLogs holds logs from both the Blockscout logs API and the RPC receipts.
		allLogs  = make(map[string][]model.BlockscoutLog)
		rpcLogs  map[string][]model.BlockscoutLog
		fetchErr error
	)

	// Launch concurrent fetches.
	g := new(errgroup.Group)

	// 1. Normal transactions.
	g.Go(func() error {
		resp, err := p.fetchBlockscoutNormalTx(address)
		if err != nil {
			return err
		}
		normalTxs = p.transformBlockscoutNormalTx(resp, address, nil)
		return nil
	})

	// 2. Token transfers.
	g.Go(func() error {
		resp, err := p.fetchBlockscoutTokenTransfers(address)
		if err != nil {
			return err
		}
		tokenTxs = p.transformBlockscoutTokenTransfers(resp, address)
		return nil
	})

	// 3. Internal transactions.
	g.Go(func() error {
		resp, err := p.fetchBlockscoutInternalTx(address)
		if err != nil {
			return err
		}
		internalTxs = p.transformBlockscoutInternalTx(resp, address)
		return nil
	})

	// 4. Logs from Blockscout “/logs” endpoint.
	g.Go(func() error {
		resp, err := p.fetchBlockscoutLogs(address)
		if err != nil {
			return err
		}
		blockscoutLogs := p.indexBlockscoutLogsByTxHash(resp)
		mergeLogMaps(allLogs, blockscoutLogs)
		return nil
	})

	// Wait for the parallel jobs to finish.
	if err := g.Wait(); err != nil {
		logger.Log.Error().Err(err).Msg("Failed fetching Blockscout data")
		return nil, err
	}

	// --------------------------------------------------------------------
	// Optional RPC receipts query (requires normalTxs + rpcURL to be present)
	// --------------------------------------------------------------------
	if len(normalTxs) > 0 && p.config.RPCURL != "" {
		blocks := make(map[int64]struct{}, len(normalTxs))
		for _, tx := range normalTxs {
			blocks[tx.Height] = struct{}{}
		}

		rpcLogs, fetchErr = p.fetchLogsByBlockFromRPC(blocks)
		if fetchErr != nil {
			// Log the error and continue using only Blockscout logs.
			logger.Log.Warn().Err(fetchErr).Msg("Failed to fetch RPC logs")
		} else {
			mergeLogMaps(allLogs, rpcLogs)
		}
	}

	// Inject logs into normal transactions (approve detection, etc.).
	if len(normalTxs) > 0 {
		normalTxs = p.transformBlockscoutNormalTxWithLogs(normalTxs, allLogs, address)
	}

	// Patch tokenTxs with gas info from normalTxs
	tokenTxs = PatchTokenTransactionsWithGasInfo(tokenTxs, normalTxs)

	// Aggregate and return all transactions.
	allTxs := append(normalTxs, tokenTxs...)
	allTxs = append(allTxs, internalTxs...)

	logger.Log.Info().
		Int("normal_count", len(normalTxs)).
		Int("token_count", len(tokenTxs)).
		Int("internal_count", len(internalTxs)).
		Int("total_transactions", len(allTxs)).
		Str("chain", p.config.ChainName).
		Str("address", address).
		Msg("Successfully fetched and merged Blockscout transactions")

	return &model.TransactionResponse{
		Result: struct {
			Transactions []model.Transaction `json:"transactions"`
		}{Transactions: allTxs},
		Id: int(p.chainID),
	}, nil
}
