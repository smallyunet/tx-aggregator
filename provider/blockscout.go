package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// TantinProvider implements the Provider interface for fetching transactions,
// token transfers, and internal transactions from the Tantin (Blockscout-like) API.
type TantinProvider struct {
	baseURL  string // Base URL for the Tantin API, e.g. "https://api.tantin.com/api/v2"
	chainID  int64  // Numeric chain ID
	chainKey string // Optional identifier for the chain, e.g. "bsc", "eth", etc.
}

// NewBlockscoutProvider creates a new BlockscoutProvider with the specified baseURL, chainID, and chainKey.
// baseURL is trimmed to remove any trailing slashes.
func NewBlockscoutProvider(baseURL string, chainID int64, chainKey string) *TantinProvider {
	logger.Log.Info().
		Str("baseURL", baseURL).
		Msg("Initializing new TantinProvider")
	return &TantinProvider{
		baseURL:  strings.TrimRight(baseURL, "/"),
		chainID:  chainID,
		chainKey: chainKey,
	}
}

// GetTransactions fetches and combines normal transactions, token transfers, and internal transactions
// from the Tantin (Blockscout-like) API for a given address.
// It uses concurrency (errgroup) to parallelize the requests, then transforms and merges the results.
func (t *TantinProvider) GetTransactions(address string) (*model.TransactionResponse, error) {
	logger.Log.Info().
		Str("chain", t.chainKey).
		Str("address", address).
		Msg("Fetching transactions from Tantin")

	var (
		normalTxs []model.Transaction
		tokenTxs  []model.Transaction
		intTxs    []model.Transaction
	)

	// Use an errgroup for concurrent requests
	g := new(errgroup.Group)

	// Fetch & transform normal transactions
	g.Go(func() error {
		respData, err := t.fetchTantinNormalTx(address)
		if err != nil {
			return err
		}
		normalTxs = t.transformTantinNormalTx(respData, address)
		return nil
	})

	// Fetch & transform token transfers
	g.Go(func() error {
		respData, err := t.fetchTantinTokenTransfers(address)
		if err != nil {
			return err
		}
		tokenTxs = t.transformTantinTokenTransfers(respData, address)
		return nil
	})

	// Fetch & transform internal transactions (currently left blank as requested)
	g.Go(func() error {
		respData, err := t.fetchTantinInternalTx(address)
		if err != nil {
			return err
		}
		intTxs = t.transformTantinInternalTx(respData, address)
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

	// Merge the results
	allTransactions := append(normalTxs, tokenTxs...)
	allTransactions = append(allTransactions, intTxs...)

	logger.Log.Info().
		Int("normal_count", len(normalTxs)).
		Int("token_count", len(tokenTxs)).
		Int("internal_count", len(intTxs)).
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

// -----------------------------------------------------------------------------
// 1. Normal Transactions
// -----------------------------------------------------------------------------

// tantinTransactionResponse represents the JSON structure returned by the
// /addresses/{address}/transactions endpoint.
type tantinTransactionResponse struct {
	Items []tantinTransaction `json:"items"`
}

// tantinTransaction represents a single transaction in the Tantin "transactions" response.
type tantinTransaction struct {
	Hash             string                 `json:"hash"`
	BlockHash        string                 `json:"block_hash"`
	BlockNumber      int64                  `json:"block_number"`
	Value            string                 `json:"value"`
	GasUsed          string                 `json:"gas_used"`
	GasLimit         string                 `json:"gas_limit"`
	GasPrice         string                 `json:"gas_price"`
	Timestamp        string                 `json:"timestamp"` // e.g. "2025-04-16T06:45:02.000000Z"
	Nonce            int64                  `json:"nonce"`
	Status           string                 `json:"status"` // "ok" for success
	Method           string                 `json:"method"`
	From             tantinAddressContainer `json:"from"`
	To               tantinAddressContainer `json:"to"`
	TransactionTypes []string               `json:"transaction_types"` // e.g. ["contract_call", "token_transfer"]
}

// tantinAddressContainer represents the "from"/"to" address object in Tantin responses.
type tantinAddressContainer struct {
	Hash string `json:"hash"`
}

// fetchTantinNormalTx retrieves normal transactions from the Tantin endpoint:
// GET /addresses/{address}/transactions
func (t *TantinProvider) fetchTantinNormalTx(address string) (*tantinTransactionResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/transactions", t.baseURL, address)
	logger.Log.Debug().Str("url", url).Msg("Fetching normal transactions from Tantin")

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch normal transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read normal transactions response: %w", err)
	}

	var result tantinTransactionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal normal transactions: %w", err)
	}

	return &result, nil
}

// transformTantinNormalTx converts the Tantin normal transactions response to []model.Transaction
func (t *TantinProvider) transformTantinNormalTx(resp *tantinTransactionResponse, address string) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No normal transactions to transform from Tantin")
		return nil
	}

	// Prepare a slice of model.Transaction
	var transactions []model.Transaction
	for _, tx := range resp.Items {
		// Convert "ok" status to integer (1 = success, 0 = fail)
		state := 0
		if strings.EqualFold(tx.Status, "ok") {
			state = 1
		}

		// Transaction direction: Outgoing by default
		tranType := model.TransTypeOut
		if strings.EqualFold(tx.To.Hash, address) {
			tranType = model.TransTypeIn
		}

		// Parse timestamp to int64 (Unix epoch)
		unixTime := parseTantinTimestampToUnix(tx.Timestamp)

		// Convert gas limit/used/price
		gasUsed := tx.GasUsed
		gasLimit := parseStringToInt64OrDefault(tx.GasLimit, 0)
		// We keep gas price as string if needed. Or parse as well if you prefer:
		// gasPriceInt := parseStringToInt64OrDefault(tx.GasPrice, 0)

		transactions = append(transactions, model.Transaction{
			ChainID:          t.chainID,
			State:            state,
			Height:           tx.BlockNumber,
			Hash:             tx.Hash,
			BlockHash:        tx.BlockHash,
			FromAddress:      tx.From.Hash,
			ToAddress:        tx.To.Hash,
			TokenAddress:     "",
			Amount:           tx.Value,    // In Wei or subunits
			GasUsed:          gasUsed,     // Keep as string or parse as int
			GasLimit:         gasLimit,    // int64
			GasPrice:         tx.GasPrice, // keep as string
			Nonce:            strconv.FormatInt(tx.Nonce, 10),
			Type:             0,                    // 0 = normal transfer
			CoinType:         model.CoinTypeNative, // 1 = native
			TokenDisplayName: "",
			Decimals:         model.NativeDefaultDecimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          "",
		})
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Transformed normal transactions from Tantin")
	return transactions
}

// -----------------------------------------------------------------------------
// 2. Token Transfers
// -----------------------------------------------------------------------------

// tantinTokenTransferResponse represents the JSON from /addresses/{address}/token-transfers
type tantinTokenTransferResponse struct {
	Items []tantinTokenTransfer `json:"items"`
}

// tantinTokenTransfer represents an individual token transfer item
type tantinTokenTransfer struct {
	BlockHash       string                 `json:"block_hash"`
	BlockNumber     int64                  `json:"block_number"`
	From            tantinAddressContainer `json:"from"`
	To              tantinAddressContainer `json:"to"`
	Timestamp       string                 `json:"timestamp"`
	TransactionHash string                 `json:"transaction_hash"`
	Token           tantinTokenInfo        `json:"token"`
	Total           tantinTokenAmount      `json:"total"`
	Type            string                 `json:"type"` // e.g. "token_minting", "token_transfer"
}

// tantinTokenInfo holds metadata about the token
type tantinTokenInfo struct {
	Address  string `json:"address"`
	Decimals string `json:"decimals"`
	IconURL  string `json:"icon_url"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
}

// tantinTokenAmount holds the decimal string representation of the transfer amount
type tantinTokenAmount struct {
	Decimals string `json:"decimals"`
	Value    string `json:"value"` // actual token amount in subunits
}

// fetchTantinTokenTransfers retrieves token transfers from Tantin:
// GET /addresses/{address}/token-transfers
func (t *TantinProvider) fetchTantinTokenTransfers(address string) (*tantinTokenTransferResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/token-transfers", t.baseURL, address)
	logger.Log.Debug().Str("url", url).Msg("Fetching token transfers from Tantin")

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token transfers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token transfers response: %w", err)
	}

	var result tantinTokenTransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token transfers: %w", err)
	}

	return &result, nil
}

// transformTantinTokenTransfers converts Tantin token transfers into []model.Transaction
func (t *TantinProvider) transformTantinTokenTransfers(resp *tantinTokenTransferResponse, address string) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No token transfers to transform from Tantin")
		return nil
	}

	var transactions []model.Transaction
	for _, tt := range resp.Items {
		// Transaction direction: Outgoing by default
		tranType := model.TransTypeOut
		if strings.EqualFold(tt.To.Hash, address) {
			tranType = model.TransTypeIn
		}

		unixTime := parseTantinTimestampToUnix(tt.Timestamp)
		// Always consider token transfers state = 1 if there's no explicit "ok"/"fail" in the data
		state := 1

		// Parse decimals from token info
		decimals := parseStringToInt64OrDefault(tt.Token.Decimals, 18) // default 18 if parse fails

		transactions = append(transactions, model.Transaction{
			ChainID:          t.chainID,
			State:            state,
			Height:           tt.BlockNumber,
			Hash:             tt.TransactionHash,
			BlockHash:        tt.BlockHash,
			FromAddress:      tt.From.Hash,
			ToAddress:        tt.To.Hash,
			TokenAddress:     tt.Token.Address,
			Amount:           tt.Total.Value,      // the raw string in subunits
			GasUsed:          "",                  // not provided in token transfers
			GasLimit:         0,                   // not provided
			GasPrice:         "",                  // not provided
			Nonce:            "",                  // not provided
			Type:             0,                   // 0 = normal, 1 = approve, etc. We'll keep default
			CoinType:         model.CoinTypeToken, // 2 = token
			TokenDisplayName: tt.Token.Name,
			Decimals:         int(decimals),
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          tt.Token.IconURL,
		})
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Transformed token transfers from Tantin")
	return transactions
}

// -----------------------------------------------------------------------------
// 3. Internal Transactions (left blank as requested)
// -----------------------------------------------------------------------------

// tantinInternalTxResponse represents the JSON from /addresses/{address}/internal-transactions
// (currently not utilized in detail)
type tantinInternalTxResponse struct {
	Items []interface{} `json:"items"`
}

// fetchTantinInternalTx retrieves internal transactions from Tantin:
// GET /addresses/{address}/internal-transactions
func (t *TantinProvider) fetchTantinInternalTx(address string) (*tantinInternalTxResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/internal-transactions", t.baseURL, address)
	logger.Log.Debug().Str("url", url).Msg("Fetching internal transactions from Tantin")

	resp, err := http.Get(url)
	if err != nil {
		// You could choose to return an empty struct or error out. We'll error for consistency.
		return nil, fmt.Errorf("failed to fetch internal transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read internal transactions response: %w", err)
	}

	var result tantinInternalTxResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal internal transactions: %w", err)
	}

	// Currently not processing these fields further (left blank)
	return &result, nil
}

// transformTantinInternalTx transforms internal transactions. For now, returns an empty slice
// as requested: "这个请求的具体处理留空".
func (t *TantinProvider) transformTantinInternalTx(_ *tantinInternalTxResponse, _ string) []model.Transaction {
	logger.Log.Warn().Msg("Internal transactions transformation is left blank by request")
	return nil
}

// -----------------------------------------------------------------------------
// Utility Functions
// -----------------------------------------------------------------------------

// parseTantinTimestampToUnix parses a timestamp like "2025-04-16T06:45:02.000000Z" into an int64 (Unix epoch)
func parseTantinTimestampToUnix(ts string) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("timestamp", ts).
			Msg("Failed to parse Tantin timestamp, returning 0")
		return 0
	}
	return parsed.Unix()
}
