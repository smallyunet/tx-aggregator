package provider

import (
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"strconv"
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
}

// NewBlockscoutProvider creates a new BlockscoutProvider with the specified baseURL, chainID, and chainKey.
// baseURL is trimmed to remove any trailing slashes.
func NewBlockscoutProvider(baseURL string, chainID int64, chainKey string) *BlockscoutProvider {
	logger.Log.Info().
		Str("baseURL", baseURL).
		Msg("Initializing new BlockscoutProvider")
	return &BlockscoutProvider{
		baseURL:  strings.TrimRight(baseURL, "/"),
		chainID:  chainID,
		chainKey: chainKey,
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
		logsResponse *tantinLogResponse

		// We will store the logs in a map keyed by transaction hash
		logsMap = make(map[string][]tantinLog)
	)

	// Use errgroup for concurrent requests
	g := new(errgroup.Group)

	// 1. Fetch & transform normal transactions
	g.Go(func() error {
		respData, err := t.fetchTantinNormalTx(address)
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
	g.Go(func() error {
		var err error
		logsResponse, err = t.fetchBlockscoutLogs(address)
		if err != nil {
			return err
		}
		logsMap = t.indexTantinLogsByTxHash(logsResponse)
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
func (t *BlockscoutProvider) fetchTantinNormalTx(address string) (*tantinTransactionResponse, error) {
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

// transformBlockscoutNormalTx is the initial conversion of Tantin normal transactions response to []model.Transaction.
// Here we do NOT yet handle "approve" detection. We simply store the base transaction data.
func (t *BlockscoutProvider) transformBlockscoutNormalTx(
	resp *tantinTransactionResponse,
	address string,
	logsMap map[string][]tantinLog, // May be nil on first pass
) []model.Transaction {

	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No normal transactions to transform from Tantin")
		return nil
	}

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
		unixTime := parseBlockscoutTimestampToUnix(tx.Timestamp)

		// Convert gas limit/used/price
		gasUsed := tx.GasUsed
		gasLimit := parseStringToInt64OrDefault(tx.GasLimit, 0)

		transactions = append(transactions, model.Transaction{
			ChainID:          t.chainID,
			State:            state,
			Height:           tx.BlockNumber,
			Hash:             tx.Hash,
			BlockHash:        tx.BlockHash,
			FromAddress:      tx.From.Hash,
			ToAddress:        tx.To.Hash,
			TokenAddress:     "",
			Amount:           tx.Value, // In Wei or subunits
			GasUsed:          gasUsed,  // Keep as string
			GasLimit:         gasLimit,
			GasPrice:         tx.GasPrice, // keep as string
			Nonce:            strconv.FormatInt(tx.Nonce, 10),
			Type:             model.TxTypeUnknown,  // 0 = normal transfer
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

	return transactions
}

// transformBlockscoutNormalTxWithLogs re-processes the already converted normal transactions
// to detect if any are "approve" type (or other ERC-20 events) by scanning the logs map.
// `logsMap` is keyed by tx hash => slice of tantinLog.
func (b *BlockscoutProvider) transformBlockscoutNormalTxWithLogs(
	normalTxs []model.Transaction,
	logsMap map[string][]tantinLog,
	address string,
) []model.Transaction {

	for i, tx := range normalTxs {
		// Does this transaction have logs in the map?
		logsForTx, found := logsMap[tx.Hash]
		if !found || len(logsForTx) == 0 {
			// No logs => nothing to detect
			continue
		}

		// We’ll see if any log indicates a recognized ERC-20 event
		var finalTxType int = model.TxTypeUnknown
		var finalTokenAddr, finalApproveVal string

		for _, lg := range logsForTx {
			// Our generic detection function:
			// DetectERC20Event(contractAddress, topics, data)
			txType, tokenAddr, approveValue := DetectERC20Event(
				lg.Address.Hash, // the contract address
				lg.Topics,
				lg.Data,
			)

			if txType != model.TxTypeUnknown {
				finalTxType = txType
				finalTokenAddr = tokenAddr
				finalApproveVal = approveValue

				// If you only want to capture the first recognized event,
				// you can break here.
				break
			}
		}

		// If we recognized an ERC-20 event, update the transaction
		if finalTxType != model.TxTypeUnknown {
			// If it’s an Approval, store the approval value
			if finalTxType == model.TxTypeApprove {
				normalTxs[i].ApproveShow = finalApproveVal
			}

			normalTxs[i].Type = finalTxType
			normalTxs[i].TokenAddress = finalTokenAddr
		}
	}

	return normalTxs
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
func (t *BlockscoutProvider) fetchTantinTokenTransfers(address string) (*tantinTokenTransferResponse, error) {
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

// transformBlockscoutTokenTransfers converts Tantin token transfers into []model.Transaction
func (t *BlockscoutProvider) transformBlockscoutTokenTransfers(resp *tantinTokenTransferResponse, address string) []model.Transaction {
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

		unixTime := parseBlockscoutTimestampToUnix(tt.Timestamp)
		// Always consider token transfers state = 1 if there's no explicit fail indicator
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
			Amount:           tt.Total.Value,       // the raw string in subunits
			GasUsed:          "",                   // not provided in token transfers
			GasLimit:         0,                    // not provided
			GasPrice:         "",                   // not provided
			Nonce:            "",                   // not provided
			Type:             model.TxTypeTransfer, // 0 = normal, 1 = approve, etc.
			CoinType:         model.CoinTypeToken,  // 2 = token
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
// 3. Internal Transactions
// -----------------------------------------------------------------------------

// tantinInternalTxResponse represents the JSON from /addresses/{address}/internal-transactions
type tantinInternalTxResponse struct {
	Items []tantinInternalTx `json:"items"`
}

// tantinInternalTx is an example structure representing a single internal transaction
type tantinInternalTx struct {
	BlockNumber     int64                 `json:"block_number"`
	CreatedContract *tantinAddressDetails `json:"created_contract"` // can be nil
	Error           string                `json:"error"`
	From            *tantinAddressDetails `json:"from"`
	To              *tantinAddressDetails `json:"to"`
	GasLimit        string                `json:"gas_limit"`
	Index           int64                 `json:"index"`
	Success         bool                  `json:"success"`
	Timestamp       string                `json:"timestamp"`
	TransactionHash string                `json:"transaction_hash"`
	Type            string                `json:"type"`  // "call", "create", etc
	Value           string                `json:"value"` // in Wei or subunits
}

// tantinAddressDetails is a more detailed address object used in internal transactions
type tantinAddressDetails struct {
	Hash               string      `json:"hash"`
	ImplementationName string      `json:"implementation_name"`
	Name               string      `json:"name"`
	EnsDomainName      string      `json:"ens_domain_name"`
	Metadata           interface{} `json:"metadata"`
	IsContract         bool        `json:"is_contract"`
	IsVerified         bool        `json:"is_verified"`
	// plus other possible fields like "private_tags", "public_tags", etc.
}

// fetchBlockscoutInternalTx retrieves internal transactions from Tantin:
// GET /addresses/{address}/internal-transactions
func (t *BlockscoutProvider) fetchBlockscoutInternalTx(address string) (*tantinInternalTxResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/internal-transactions", t.baseURL, address)
	logger.Log.Debug().Str("url", url).Msg("Fetching internal transactions from Tantin")

	resp, err := http.Get(url)
	if err != nil {
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

	return &result, nil
}

// transformBlockscoutInternalTx converts internal transaction data into []model.Transaction.
// If you only want to store minimal details, this is an example approach.
func (t *BlockscoutProvider) transformBlockscoutInternalTx(resp *tantinInternalTxResponse, address string) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No internal transactions to transform from Tantin")
		return nil
	}

	var transactions []model.Transaction
	for _, itx := range resp.Items {
		// For internal tx success/fail
		state := 0
		if itx.Success {
			state = 1
		}

		unixTime := parseBlockscoutTimestampToUnix(itx.Timestamp)

		fromHash := ""
		toHash := ""
		if itx.From != nil {
			fromHash = itx.From.Hash
		}
		if itx.To != nil {
			toHash = itx.To.Hash
		}

		// Transaction direction: Outgoing by default
		tranType := model.TransTypeOut
		if strings.EqualFold(toHash, address) {
			tranType = model.TransTypeIn
		}

		gasLimit := parseStringToInt64OrDefault(itx.GasLimit, 0)

		transactions = append(transactions, model.Transaction{
			ChainID:          t.chainID,
			State:            state,
			Height:           itx.BlockNumber,
			Hash:             itx.TransactionHash, // internal tx uses outer transaction's hash
			BlockHash:        "",                  // not provided by Tantin for internal
			FromAddress:      fromHash,
			ToAddress:        toHash,
			TokenAddress:     "",
			Amount:           itx.Value,
			GasUsed:          "", // not provided in internal tx
			GasLimit:         gasLimit,
			GasPrice:         "",
			Nonce:            "",
			Type:             model.TxTypeInternal, // can define a custom type code if desired
			CoinType:         model.CoinTypeNative, // typically native if transferring
			TokenDisplayName: "",
			Decimals:         model.NativeDefaultDecimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          "",
		})
	}
	return transactions
}

// -----------------------------------------------------------------------------
// 4. Logs (used for detecting "approve" events, etc.)
// -----------------------------------------------------------------------------

// tantinLogResponse represents the JSON from /addresses/{address}/logs
type tantinLogResponse struct {
	Items []tantinLog `json:"items"`
}

// tantinLog is a minimal representation of a log item. You can expand as needed.
type tantinLog struct {
	Address         tantinAddressDetails `json:"address"`
	BlockHash       string               `json:"block_hash"`
	BlockNumber     int64                `json:"block_number"`
	Data            string               `json:"data"`
	Decoded         *tantinLogDecoded    `json:"decoded"` // can be nil
	Index           int64                `json:"index"`
	SmartContract   tantinAddressDetails `json:"smart_contract"`
	Topics          []string             `json:"topics"`
	TransactionHash string               `json:"transaction_hash"`
}

// tantinLogDecoded holds decoded function/event data if available
type tantinLogDecoded struct {
	MethodCall string `json:"method_call"`
	MethodID   string `json:"method_id"`
	// Possibly an array of parameters, etc.
	Parameters []struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Value   string `json:"value"`
		Indexed bool   `json:"indexed"`
	} `json:"parameters"`
}

// fetchBlockscoutLogs retrieves logs from Tantin:
// GET /addresses/{address}/logs
func (t *BlockscoutProvider) fetchBlockscoutLogs(address string) (*tantinLogResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/logs", t.baseURL, address)
	logger.Log.Debug().Str("url", url).Msg("Fetching logs from Tantin")

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs response: %w", err)
	}

	var result tantinLogResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %w", err)
	}

	return &result, nil
}

// indexTantinLogsByTxHash stores each log in a map keyed by transaction hash.
func (t *BlockscoutProvider) indexTantinLogsByTxHash(resp *tantinLogResponse) map[string][]tantinLog {
	logsMap := make(map[string][]tantinLog)
	if resp == nil || len(resp.Items) == 0 {
		return logsMap
	}

	for _, lg := range resp.Items {
		txHash := lg.TransactionHash
		logsMap[txHash] = append(logsMap[txHash], lg)
	}
	return logsMap
}
