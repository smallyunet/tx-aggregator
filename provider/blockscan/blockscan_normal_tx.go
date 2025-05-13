package blockscan

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// This file contains functions for fetching and transforming normal transactions from Blockscan API
// (such as Etherscan, BscScan, etc.)

// fetchNormalTx retrieves normal transactions for a given address from the Blockscan API.
// It constructs the API request with parameters from the provider configuration.
//
// Parameters:
//   - addr: The blockchain address to fetch transactions for
//
// Returns:
//   - *types.BlockscanNormalTxResp: The API response containing transaction data
//   - error: Any error encountered during the API request
func (p *BlockscanProvider) fetchNormalTx(addr string) (*types.BlockscanNormalTxResp, error) {
	// Construct query parameters for the Blockscan API request
	q := url.Values{
		"module":     {"account"},
		"action":     {"txlist"},
		"address":    {addr},
		"startblock": {strconv.FormatInt(p.cfg.Startblock, 10)},
		"endblock":   {strconv.FormatInt(p.cfg.Endblock, 10)},
		"page":       {strconv.FormatInt(p.cfg.Page, 10)},
		"offset":     {fmt.Sprint(p.cfg.RequestPageSize)},
		"sort":       {p.cfg.Sort},
		"apikey":     {p.cfg.APIKey},
	}
	var out types.BlockscanNormalTxResp

	// Build the complete URL with query parameters
	u := fmt.Sprintf("%s?%s", p.cfg.URL, q.Encode())

	// Execute the HTTP request with logging
	if err := utils.DoHttpRequestWithLogging("GET", "blockscan.normalTx", u, nil, nil, &out); err != nil {
		return nil, err
	}

	// Check if the API returned an error status
	if out.Status == types.StatusError {
		logger.Log.Warn().
			Str("error_message", out.Message).
			Str("address", addr).
			Msg("Failed to fetch normal transactions from Blockscan")
		return nil, fmt.Errorf("blockscan error: %s", out.Message)
	}

	return &out, nil
}

// transformNormalTx converts the Blockscan API response into a standardized Transaction format.
// It processes each transaction in the response and extracts relevant information.
//
// Parameters:
//   - resp: The API response containing transaction data
//   - address: The blockchain address used for determining transaction direction (in/out)
//
// Returns:
//   - []types.Transaction: A slice of standardized Transaction objects
func (p *BlockscanProvider) transformNormalTx(resp *types.BlockscanNormalTxResp, address string) []types.Transaction {
	// Validate response data
	if resp == nil || resp.Status != types.StatusOK || len(resp.Result) == 0 {
		return nil
	}

	var txs []types.Transaction
	// Process each transaction in the response
	for _, it := range resp.Result {
		// Parse numeric values from string representations
		height := utils.ParseStringToInt64OrDefault(it.BlockNumber, 0)
		unixTime := utils.ParseStringToInt64OrDefault(it.TimeStamp, 0)
		txIndex := utils.ParseStringToInt64OrDefault(it.TransactionIndex, 0)

		// Determine transaction state (success or failure)
		state := types.TxStateFail
		if it.IsError == "0" && it.TxReceiptStatus == "1" {
			state = types.TxStateSuccess
		}

		// Determine transaction direction (incoming or outgoing)
		tranType := types.TransTypeOut
		if strings.EqualFold(it.To, address) {
			tranType = types.TransTypeIn
		}

		// Parse and normalize transaction values
		amountRaw, _ := utils.NormalizeNumericString(it.Value)
		amount := utils.DivideByDecimals(amountRaw, types.NativeDefaultDecimals)
		gasLimit, _ := utils.NormalizeNumericString(it.Gas)
		gasUsed, _ := utils.NormalizeNumericString(it.GasUsed)
		gasPrice, _ := utils.NormalizeNumericString(it.GasPrice)
		nonce, _ := utils.NormalizeNumericString(it.Nonce)

		// Get native token symbol for the current chain
		nativeSymbol, err := utils.NativeTokenByChainID(p.chainID)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Int64("chain_id", p.chainID).
				Msg("Failed to get native token name")
		}

		// Create standardized transaction object and add to result list
		txs = append(txs, types.Transaction{
			ChainID:          p.chainID,
			State:            state,
			Height:           height,
			Hash:             it.Hash,
			BlockHash:        it.BlockHash,
			TxIndex:          txIndex,
			FromAddress:      it.From,
			ToAddress:        it.To,
			TokenAddress:     "",
			Balance:          amountRaw,
			Amount:           amount,
			GasLimit:         gasLimit,
			GasUsed:          gasUsed,
			GasPrice:         gasPrice,
			Nonce:            nonce,
			Type:             types.TxTypeUnknown, // native transfer
			CoinType:         types.CoinTypeNative,
			TokenDisplayName: nativeSymbol,
			Decimals:         types.NativeDefaultDecimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
		})
	}
	return txs
}
