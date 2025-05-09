package ankr

import (
	"fmt"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/provider"
	"tx-aggregator/types"
	"tx-aggregator/utils"

	"golang.org/x/sync/errgroup"
)

// AnkrProvider implements the Provider interface for interacting with Ankr's blockchain API
// It handles fetching and processing both native token transactions and token transfers
var _ provider.Provider = (*AnkrProvider)(nil)

// AnkrProvider provides methods to interact with the Ankr API
type AnkrProvider struct {
	apiKey string // API key for authentication
	url    string // Base URL for API requests
}

// NewAnkrProvider creates a new AnkrProvider instance with the given API key and URL
// The URL is trimmed to remove any trailing slashes
func NewAnkrProvider(apiKey, url string) *AnkrProvider {
	logger.Log.Info().Str("url", url).Msg("Initializing new AnkrProvider")
	return &AnkrProvider{
		apiKey: apiKey,
		url:    strings.TrimRight(url, "/"),
	}
}

// GetTransactions fetches and transforms both normal transactions and token transfers for the given address,
// using concurrency in a more streamlined way (fetch & transform in the same goroutine).
func (a *AnkrProvider) GetTransactions(params *types.TransactionQueryParams) (*types.TransactionResponse, error) {
	address := params.Address

	logger.Log.Info().
		Str("address", address).
		Strs("params_chainnames", params.ChainNames).
		Msg("Starting to fetch all transactions for address")

	var (
		normalTxs []types.Transaction
		tokenTxs  []types.Transaction
	)

	// Use an errgroup to concurrently fetch and transform both types of transactions
	g := new(errgroup.Group)

	// Concurrently fetch and transform normal transactions
	g.Go(func() error {
		normalTxResp, err := a.GetTransactionsByAddress(params)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("address", address).
				Msg("Failed to fetch normal transactions")
			return fmt.Errorf("failed to get normal transactions: %w", err)
		}
		// Transform directly at this step
		normalTxs = a.transformAnkrNormalTx(normalTxResp, address)
		return nil
	})

	// Concurrently fetch and transform token transfers
	g.Go(func() error {
		tokenTransferResp, err := a.GetTokenTransfers(params)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("address", address).
				Msg("Failed to fetch token transfers")
			return fmt.Errorf("failed to get token transfers: %w", err)
		}
		// Transform directly at this step
		tokenTxs = a.transformAnkrTokenTransfers(tokenTransferResp, address)
		return nil
	})

	// Wait for both concurrent tasks to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Patch token transfers using matching normal transactions
	tokenTxs = utils.PatchTokenTransactionsWithNormalTxInfo(tokenTxs, normalTxs)

	// Merge the final results
	transactions := append(normalTxs, tokenTxs...)

	logger.Log.Info().
		Str("address", address).
		Int("normal_txs_count", len(normalTxs)).
		Int("token_transfers_count", len(tokenTxs)).
		Int("total_transactions", len(transactions)).
		Msg("Successfully fetched and processed all transactions")

	return &types.TransactionResponse{
		Result: struct {
			Transactions []types.Transaction `json:"transactions"`
		}{
			Transactions: transactions,
		},
	}, nil
}

func (p *AnkrProvider) sendRequest(requestBody interface{}, result interface{}, label string) error {
	fullURL := fmt.Sprintf("%s/%s", p.url, p.apiKey)
	return utils.DoHttpRequestWithLogging("POST", "ankr."+label, fullURL, requestBody, map[string]string{
		"Content-Type": "application/json",
		"x-api-key":    p.apiKey,
	}, result)
}
