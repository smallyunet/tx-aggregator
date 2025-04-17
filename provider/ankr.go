package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// AnkrProvider implements the Provider interface for interacting with Ankr's blockchain API
// It handles fetching and processing both native token transactions and token transfers
var _ Provider = (*AnkrProvider)(nil)

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
func (a *AnkrProvider) GetTransactions(address string) (*model.TransactionResponse, error) {
	logger.Log.Info().
		Str("address", address).
		Msg("Starting to fetch all transactions for address")

	var (
		normalTxs []model.Transaction
		tokenTxs  []model.Transaction
	)

	// Use an errgroup to concurrently fetch and transform both types of transactions
	g := new(errgroup.Group)

	// Concurrently fetch and transform normal transactions
	g.Go(func() error {
		normalTxResp, err := a.GetTransactionsByAddress(address)
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
		tokenTransferResp, err := a.GetTokenTransfers(address)
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
	tokenTxs = PatchTokenTransactionsWithGasInfo(tokenTxs, normalTxs)

	// Merge the final results
	transactions := append(normalTxs, tokenTxs...)

	logger.Log.Info().
		Str("address", address).
		Int("normal_txs_count", len(normalTxs)).
		Int("token_transfers_count", len(tokenTxs)).
		Int("total_transactions", len(transactions)).
		Msg("Successfully fetched and processed all transactions")

	return &model.TransactionResponse{
		Result: struct {
			Transactions []model.Transaction `json:"transactions"`
		}{
			Transactions: transactions,
		},
		Id: 1,
	}, nil
}

// sendRequest sends a POST request to the Ankr API and decodes the JSON response
// It handles authentication, request formatting, and error handling
func (p *AnkrProvider) sendRequest(requestBody interface{}, result interface{}) error {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to marshal request body")
		return fmt.Errorf("marshal request failed: %w", err)
	}

	fullURL := fmt.Sprintf("%s/%s", p.url, p.apiKey)
	logger.Log.Debug().
		Str("url", fullURL).
		Msg("Sending request to Ankr API")

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to create HTTP request")
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to send request to Ankr API")
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log.Error().
			Int("status_code", resp.StatusCode).
			Msg("Ankr API returned non-success status code")
		return fmt.Errorf("ankr api responded with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to read response body")
		return fmt.Errorf("read response failed: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Failed to unmarshal response body")
		return fmt.Errorf("unmarshal response failed: %w", err)
	}

	return nil
}
