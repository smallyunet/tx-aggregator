package blockscan

// This file contains functions for fetching and processing internal transactions from Blockscan API.
// Internal transactions are transactions that are created by smart contract execution, not directly by users.

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// fetchInternalTx retrieves internal transactions for a specific address from the Blockscan API.
// It constructs a query with parameters like address, block range, pagination settings, and API key.
// Returns the API response containing internal transactions or an error if the request fails.
func (p *BlockscanProvider) fetchInternalTx(addr string) (*types.BlockscanInternalTxResp, error) {
	// Construct query parameters for the Blockscan API request
	q := url.Values{
		"module":     {"account"},
		"action":     {"txlistinternal"},
		"address":    {addr},
		"startblock": {strconv.FormatInt(p.cfg.Startblock, 10)},
		"endblock":   {strconv.FormatInt(p.cfg.Endblock, 10)},
		"page":       {strconv.FormatInt(p.cfg.Page, 10)},
		"offset":     {fmt.Sprint(p.cfg.RequestPageSize)},
		"sort":       {p.cfg.Sort},
		"apikey":     {p.cfg.APIKey},
	}
	var out types.BlockscanInternalTxResp
	// Construct the full URL with query parameters and make the HTTP request
	u := fmt.Sprintf("%s?%s", p.cfg.URL, q.Encode())
	if err := utils.DoHttpRequestWithLogging("GET", "blockscan.internalTx", u, nil, nil, &out); err != nil {
		return nil, err
	}

	// Check if the API returned an error status and log the error
	if out.Status == types.StatusError {
		logger.Log.Warn().
			Str("error_message", out.Message).
			Str("address", addr).
			Msg("Failed to fetch internal transactions from Blockscan")
		return nil, fmt.Errorf("blockscan error: %s", out.Message)
	}

	return &out, nil
}

// transformInternalTx converts the raw Blockscan API response into a standardized Transaction format.
// It processes each internal transaction in the response, extracting relevant fields and normalizing values.
// The function determines transaction state (success/fail), transaction type (in/out), and converts numeric values.
// Returns an array of Transaction objects or nil if the response is empty or invalid.
func (p *BlockscanProvider) transformInternalTx(resp *types.BlockscanInternalTxResp, addr string) []types.Transaction {
	// Return nil if response is invalid or empty
	if resp == nil || resp.Status != types.StatusOK || len(resp.Result) == 0 {
		return nil
	}

	var txs []types.Transaction
	for _, it := range resp.Result {
		// Parse block height and timestamp
		height := utils.ParseStringToInt64OrDefault(it.BlockNumber, 0)
		unixTime := utils.ParseStringToInt64OrDefault(it.TimeStamp, 0)

		// Determine transaction state (success or fail)
		// IsError="0" means the transaction was successful
		state := types.TxStateFail
		if it.IsError == "0" {
			state = types.TxStateSuccess
		}

		// Determine transaction type (incoming or outgoing)
		// If the transaction's recipient matches the address we're querying, it's an incoming transaction
		tranType := types.TransTypeOut
		if strings.EqualFold(it.To, addr) {
			tranType = types.TransTypeIn
		}

		// Normalize numeric values (value, gas limit, gas used)
		valueRaw, _ := utils.NormalizeNumericString(it.Value)
		value := utils.DivideByDecimals(valueRaw, types.NativeDefaultDecimals)
		gasLimit, _ := utils.NormalizeNumericString(it.Gas)
		gasUsed, _ := utils.NormalizeNumericString(it.GasUsed)

		// Construct a standardized Transaction object with all the processed data
		txs = append(txs, types.Transaction{
			ChainID:      p.chainID,
			State:        state,
			Height:       height,
			Hash:         it.Hash,
			FromAddress:  it.From,
			ToAddress:    it.To,
			Balance:      valueRaw,
			Amount:       value,
			GasLimit:     gasLimit,
			GasUsed:      gasUsed,
			Type:         types.TxTypeInternal,
			CoinType:     types.CoinTypeInternal,
			Decimals:     types.NativeDefaultDecimals,
			CreatedTime:  unixTime,
			ModifiedTime: unixTime,
			TranType:     tranType,
		})
	}
	// Return the array of standardized Transaction objects
	return txs
}
