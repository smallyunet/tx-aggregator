// File: provider/blockscout_provider.go
// Package provider implements data sources for the transaction‑aggregator service.
// The BlockscoutProvider fetches transactions, token transfers, internal
// transactions, and logs from a Blockscout‑compatible REST API, and (optionally)
// extra logs from an RPC endpoint. All comments are in English as requested.

package blockscout

import (
	"fmt"
	"io"
	"net/http"
	"time"
	"tx-aggregator/logger"
	"tx-aggregator/types"
	"tx-aggregator/utils"

	"golang.org/x/sync/errgroup"
)

// BlockscoutProvider implements the Provider interface for fetching transaction
// data from a Blockscout‑compatible API.
type BlockscoutProvider struct {
	chainID int64 // Numeric chain ID
	config  types.BlockscoutConfig
}

// NewBlockscoutProvider returns a new BlockscoutProvider.
// Trailing slashes are trimmed from baseURL for consistency.
func NewBlockscoutProvider(chainID int64, config types.BlockscoutConfig) *BlockscoutProvider {
	logger.Log.Info().
		Msg("Initializing BlockscoutProvider")

	return &BlockscoutProvider{
		chainID: chainID,
		config:  config,
	}
}

// GetTransactions concurrently fetches all relevant data for a single address
// and returns a unified TransactionResponse.
func (p *BlockscoutProvider) GetTransactions(address string) (*types.TransactionResponse, error) {
	logger.Log.Info().
		Str("chain", p.config.ChainName).
		Str("address", address).
		Msg("Fetching transactions from Blockscout")

	var (
		normalTxs   []types.Transaction
		tokenTxs    []types.Transaction
		internalTxs []types.Transaction

		// allLogs holds logs from both the Blockscout logs API and the RPC receipts.
		allLogs  = make(map[string][]types.BlockscoutLog)
		rpcLogs  map[string][]types.BlockscoutLog
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
		utils.MergeLogMaps(allLogs, blockscoutLogs)
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
			utils.MergeLogMaps(allLogs, rpcLogs)
		}
	}

	// Inject logs into normal transactions (approve detection, etc.).
	if len(normalTxs) > 0 {
		normalTxs = p.transformBlockscoutNormalTxWithLogs(normalTxs, allLogs, address)
	}

	// Patch tokenTxs with gas info from normalTxs
	tokenTxs = utils.PatchTokenTransactionsWithNormalTxInfo(tokenTxs, normalTxs)

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

	return &types.TransactionResponse{
		Result: struct {
			Transactions []types.Transaction `json:"transactions"`
		}{Transactions: allTxs},
		Id: int(p.chainID),
	}, nil
}

// doLoggedHttpGet sends a GET request to the given URL, logs duration and errors, and returns the response body.
func doLoggedHttpGet(label string, url string) ([]byte, error) {
	start := time.Now()
	resp, err := http.Get(url)
	duration := time.Since(start)

	if err != nil {
		logger.Log.Error().
			Str("label", label).
			Str("url", url).
			Dur("duration", duration).
			Err(err).
			Msg("Failed to send GET request")
		return nil, fmt.Errorf("http GET failed for %s: %w", label, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error().
			Str("label", label).
			Str("url", url).
			Dur("duration", duration).
			Err(err).
			Msg("Failed to read response body")
		return nil, fmt.Errorf("read body failed for %s: %w", label, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log.Error().
			Str("label", label).
			Str("url", url).
			Int("status_code", resp.StatusCode).
			Dur("duration", duration).
			Msg("Non-200 response")
		return nil, fmt.Errorf("non-200 status for %s: %d", label, resp.StatusCode)
	}

	logger.Log.Info().
		Str("label", label).
		Str("url", url).
		Int("status_code", resp.StatusCode).
		Int("response_size", len(body)).
		Dur("duration", duration).
		Msg("Successful GET request")

	return body, nil
}
