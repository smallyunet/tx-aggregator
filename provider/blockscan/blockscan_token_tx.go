// Package blockscan provides functionality to interact with Blockscan API
// for retrieving blockchain transaction data including token transactions.
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

// fetchTokenTx retrieves token transactions for a specific address from the Blockscan API.
// It constructs the API request with appropriate parameters and handles error responses.
//
// Parameters:
//   - addr: The blockchain address to fetch token transactions for
//
// Returns:
//   - *types.BlockscanTokenTxResp: The API response containing token transactions
//   - error: Any error encountered during the API request
func (p *BlockscanProvider) fetchTokenTx(addr string) (*types.BlockscanTokenTxResp, error) {
	// Prepare query parameters for the Blockscan API request
	q := url.Values{
		"module":  {"account"},             // Specify the module as account
		"action":  {"tokentx"},             // Request token transactions
		"address": {addr},                  // The address to query transactions for
		"page":    {strconv.FormatInt(p.cfg.Page, 10)}, // Pagination parameter
		"offset":  {fmt.Sprint(p.cfg.RequestPageSize)},  // Number of results per page
		"sort":    {p.cfg.Sort},            // Sorting order (asc/desc)
		"apikey":  {p.cfg.APIKey},          // API key for authentication
	}
	// Prepare response variable to store API results
	var out types.BlockscanTokenTxResp

	// Construct the full URL with query parameters
	u := fmt.Sprintf("%s?%s", p.cfg.URL, q.Encode())

	// Execute HTTP GET request with logging
	if err := utils.DoHttpRequestWithLogging("GET", "blockscan.tokenTx", u, nil, nil, &out); err != nil {
		return nil, err
	}

	// Check if the API returned an error status
	if out.Status == types.StatusError {
		// Log the error with relevant details
		logger.Log.Warn().
			Str("error_message", out.Message).
			Str("address", addr).
			Msg("Failed to fetch token transactions from Blockscan")
		return nil, fmt.Errorf("blockscan error: %s", out.Message)
	}

	// Return successful response
	return &out, nil
}

// transformTokenTx converts the Blockscan API response into a standardized transaction format.
// It processes each token transaction and extracts relevant information such as amounts,
// addresses, gas details, and transaction types.
//
// Parameters:
//   - resp: The API response containing token transactions
//   - addr: The address used to determine transaction direction (in/out)
//
// Returns:
//   - []types.Transaction: A slice of standardized transaction objects
func (p *BlockscanProvider) transformTokenTx(resp *types.BlockscanTokenTxResp, addr string) []types.Transaction {
	// Return nil if response is invalid or empty
	if resp == nil || resp.Status != types.StatusOK || len(resp.Result) == 0 {
		return nil
	}

	// Initialize slice to store transformed transactions
	var txs []types.Transaction

	// Process each token transaction in the response
	for _, tt := range resp.Result {
		// Parse numeric string values to int64
		height := utils.ParseStringToInt64OrDefault(tt.BlockNumber, 0)
		unixTime := utils.ParseStringToInt64OrDefault(tt.TimeStamp, 0)
		txIndex := utils.ParseStringToInt64OrDefault(tt.TransactionIndex, 0)
		decimals := utils.ParseStringToInt64OrDefault(tt.TokenDecimal, types.NativeDefaultDecimals)

		// Normalize and format token amount values
		balanceRaw, _ := utils.NormalizeNumericString(tt.Value)
		amount := utils.DivideByDecimals(balanceRaw, int(decimals))

		// Determine transaction direction (in/out) based on the address
		tranType := types.TransTypeOut
		if strings.EqualFold(tt.To, addr) {
			tranType = types.TransTypeIn
		}

		// Process gas-related values
		gasLimit, _ := utils.NormalizeNumericString(tt.Gas)
		gasUsed, _ := utils.NormalizeNumericString(tt.GasUsed)
		gasPrice, _ := utils.NormalizeNumericString(tt.GasPrice)

		// Create standardized transaction object and add to result slice
		txs = append(txs, types.Transaction{
			ChainID:          p.chainID,
			Height:           height,
			Hash:             tt.Hash,
			BlockHash:        tt.BlockHash,
			TxIndex:          txIndex,
			FromAddress:      tt.From,
			ToAddress:        tt.To,
			TokenAddress:     tt.ContractAddress,
			Balance:          balanceRaw,
			Amount:           amount,
			GasLimit:         gasLimit,
			GasUsed:          gasUsed,
			GasPrice:         gasPrice,
			Type:             types.TxTypeTransfer,
			CoinType:         types.CoinTypeToken,
			TokenDisplayName: tt.TokenSymbol,
			Decimals:         decimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
		})
	}

	// Return the transformed transactions
	return txs
}
