package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"tx-aggregator/config"

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

// GetTransactions combines both normal transactions and token transfers for the given address
// It fetches data from Ankr API and transforms it into a unified transaction list
func (a *AnkrProvider) GetTransactions(address string) (*model.TransactionResponse, error) {
	logger.Log.Info().
		Str("address", address).
		Msg("Starting to fetch all transactions for address")

	// 1. Fetch normal transactions (native token transfers)
	normalTxResp, err := a.GetTransactionsByAddress(address)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch normal transactions")
		return nil, fmt.Errorf("failed to get normal transactions: %w", err)
	}

	// 2. Fetch token transfers (ERC20/BEP20/etc)
	tokenTransferResp, err := a.GetTokenTransfers(address)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch token transfers")
		return nil, fmt.Errorf("failed to get token transfers: %w", err)
	}

	// 3. Transform both responses into a unified transaction list
	normalTxs := a.transformNormalTransactions(normalTxResp, address)
	tokenTxs := a.transformTokenTransfers(tokenTransferResp, address)

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

// GetTransactionsByAddress retrieves normal transactions from Ankr for the given address
// These are native token transfers (ETH, BNB, MATIC, etc.)
func (p *AnkrProvider) GetTransactionsByAddress(address string) (*model.AnkrTransactionResponse, error) {
	logger.Log.Debug().
		Str("address", address).
		Msg("Fetching normal transactions from Ankr")

	requestBody := model.AnkrTransactionRequest{
		JSONRPC: "2.0",
		Method:  "ankr_getTransactionsByAddress",
		Params: map[string]interface{}{
			"blockchain":  config.AppConfig.Ankr.RequestBlockchains,
			"includeLogs": true,
			"descOrder":   true,
			"pageSize":    config.AppConfig.Ankr.RequestPageSize,
			"address":     address,
		},
		ID: 1,
	}

	var result model.AnkrTransactionResponse
	if err := p.sendRequest(requestBody, &result); err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch normal transactions from Ankr")
		return nil, err
	}

	logger.Log.Debug().
		Str("address", address).
		Int("tx_count", len(result.Result.Transactions)).
		Msg("Successfully fetched normal transactions")
	return &result, nil
}

// GetTokenTransfers retrieves token transfer events from Ankr for the given address
// These are ERC20/BEP20/etc token transfers
func (p *AnkrProvider) GetTokenTransfers(address string) (*model.AnkrTokenTransferResponse, error) {
	logger.Log.Debug().
		Str("address", address).
		Msg("Fetching token transfers from Ankr")

	requestBody := model.AnkrTransactionRequest{
		JSONRPC: "2.0",
		Method:  "ankr_getTokenTransfers",
		Params: map[string]interface{}{
			"blockchain": config.AppConfig.Ankr.RequestBlockchains,
			"pageSize":   config.AppConfig.Ankr.RequestPageSize,
			"address":    address,
		},
		ID: 1,
	}

	var result model.AnkrTokenTransferResponse
	if err := p.sendRequest(requestBody, &result); err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch token transfers from Ankr")
		return nil, err
	}

	logger.Log.Debug().
		Str("address", address).
		Int("transfer_count", len(result.Result.Transfers)).
		Msg("Successfully fetched token transfers")
	return &result, nil
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

// transformNormalTransactions converts AnkrTransactionResponse into a slice of model.Transaction
// These are native token transfers (ETH, BNB, MATIC, etc.)
func (a *AnkrProvider) transformNormalTransactions(resp *model.AnkrTransactionResponse, address string) []model.Transaction {
	if resp == nil || resp.Result.Transactions == nil {
		logger.Log.Warn().
			Msg("No normal transactions to transform")
		return nil
	}

	logger.Log.Debug().
		Int("tx_count", len(resp.Result.Transactions)).
		Msg("Transforming normal transactions")

	var transactions []model.Transaction
	for _, tx := range resp.Result.Transactions {
		chainID := config.ChainIDByName(tx.Blockchain)
		height := parseStringToInt64OrDefault(tx.BlockNumber, 0)
		states := parseStringToInt64OrDefault(tx.Status, 0)
		timestamp := parseStringToInt64OrDefault(tx.Timestamp, 0)
		gasLimit := parseStringToInt64OrDefault(tx.Gas, 0)

		// Detect ERC20 type and approve value from transaction logs
		txType, tokenAddr, approveValue := DetectERC20Type(tx.Logs)

		approveShow := ""
		iconURL := ""

		if txType != -1 {
			if txType == model.TxTypeApprove {
				approveShow = approveValue // Directly assign hex string, e.g., "0x000000...0001"
			}
		}

		tranType := model.TransTypeOut // default to outgoing
		if strings.EqualFold(tx.To, address) {
			tranType = model.TransTypeIn
		}

		transactions = append(transactions, model.Transaction{
			ChainID:          chainID,
			TokenID:          0,
			State:            int(states),
			Height:           height,
			Hash:             tx.Hash,
			BlockHash:        tx.BlockHash,
			FromAddress:      tx.From,
			ToAddress:        tx.To,
			TokenAddress:     tokenAddr,
			Amount:           tx.Value,
			GasUsed:          tx.GasUsed,
			GasLimit:         gasLimit,
			GasPrice:         tx.GasPrice,
			Nonce:            tx.Nonce,
			Type:             txType,
			CoinType:         model.CoinTypeNative,
			TokenDisplayName: "",
			Decimals:         model.NativeDefaultDecimals,
			CreatedTime:      timestamp,
			ModifiedTime:     timestamp,
			TranType:         tranType,
			ApproveShow:      approveShow,
			IconURL:          iconURL,
		})
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Successfully transformed normal transactions")
	return transactions
}

// transformTokenTransfers converts AnkrTokenTransferResponse into a slice of model.Transaction
// These represent ERC20/BEP20/etc token transfers
func (a *AnkrProvider) transformTokenTransfers(resp *model.AnkrTokenTransferResponse, address string) []model.Transaction {
	if resp == nil || resp.Result.Transfers == nil {
		logger.Log.Warn().
			Msg("No token transfers to transform")
		return nil
	}

	logger.Log.Debug().
		Int("transfer_count", len(resp.Result.Transfers)).
		Msg("Transforming token transfers")

	var transactions []model.Transaction
	for _, tr := range resp.Result.Transfers {
		chainID := config.ChainIDByName(tr.Blockchain)

		tranType := model.TransTypeOut // default to outgoing
		if strings.EqualFold(tr.ToAddress, address) {
			tranType = model.TransTypeIn
		}

		transactions = append(transactions, model.Transaction{
			ChainID:          chainID,
			TokenID:          0,
			State:            1,
			Height:           int64(tr.BlockHeight),
			Hash:             tr.TransactionHash,
			BlockHash:        "", // not provided by transfer API
			FromAddress:      tr.FromAddress,
			ToAddress:        tr.ToAddress,
			TokenAddress:     tr.ContractAddress,
			Amount:           tr.Value,
			GasUsed:          "",                  // not provided by transfer API
			GasLimit:         0,                   // not available
			GasPrice:         "",                  // not available
			Nonce:            "",                  // not available
			Type:             0,                   // default to transfer
			CoinType:         model.CoinTypeToken, // 2 = token
			TokenDisplayName: tr.TokenName,
			Decimals:         tr.TokenDecimals,
			CreatedTime:      tr.Timestamp,
			ModifiedTime:     tr.Timestamp,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          tr.Thumbnail, // optional logo/image
		})
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Successfully transformed token transfers")
	return transactions
}

// parseStringToInt64OrDefault converts a string to int64, supporting hex with "0x" prefix
// Returns the default value if parsing fails
func parseStringToInt64OrDefault(s string, def int64) int64 {
	var val int64
	var err error

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		val, err = strconv.ParseInt(s[2:], 16, 64)
	} else {
		val, err = strconv.ParseInt(s, 10, 64)
	}

	if err != nil {
		logger.Log.Warn().
			Err(err).
			Str("input", s).
			Int64("default", def).
			Msg("Failed to parse string to int64, using default value")
		return def
	}
	return val
}
