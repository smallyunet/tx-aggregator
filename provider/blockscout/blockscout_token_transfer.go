package blockscout

import (
	"fmt"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// fetchBlockscoutTokenTransfers retrieves token transfers from Blockscout:
// GET /addresses/{address}/token-transfers
func (t *BlockscoutProvider) fetchBlockscoutTokenTransfers(address string) (*types.BlockscoutTokenTransferResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/token-transfers?limit=%d", t.config.URL, address, t.config.RequestPageSize)
	var result types.BlockscoutTokenTransferResponse
	if err := utils.DoHttpRequestWithLogging("GET", "blockscout.tokenTransfers", url, nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// / transformBlockscoutTokenTransfers converts Blockscout token transfers into []model.Transaction.
func (t *BlockscoutProvider) transformBlockscoutTokenTransfers(
	resp *types.BlockscoutTokenTransferResponse,
	address string,
) []types.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No token transfers to transform from Blockscout")
		return nil
	}

	var transactions []types.Transaction

	for _, tt := range resp.Items {
		// Determine transaction direction
		tranType := types.TransTypeOut
		if strings.EqualFold(tt.To.Hash, address) {
			tranType = types.TransTypeIn
		}

		// Parse timestamp and decimals
		unixTime := utils.ParseBlockscoutTimestampToUnix(tt.Timestamp)
		decimals := utils.ParseStringToInt64OrDefault(tt.Token.Decimals, types.NativeDefaultDecimals) // Default to 18 if missing
		amountRaw, err := utils.NormalizeNumericString(tt.Total.Value)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("address", address).
				Msg("Failed to normalize token transfer amount")
		}
		amount := utils.DivideByDecimals(amountRaw, int(decimals))

		// Build transaction object
		transaction := types.Transaction{
			ChainID:          t.chainID,
			TokenID:          0,
			State:            types.TxStateSuccess, // Token transfers are assumed successful
			Height:           tt.BlockNumber,
			Hash:             tt.TransactionHash,
			BlockHash:        tt.BlockHash,
			FromAddress:      tt.From.Hash,
			ToAddress:        tt.To.Hash,
			TokenAddress:     tt.Token.Address,
			Balance:          amountRaw,
			Amount:           amount,
			GasUsed:          "",                   // Not provided
			GasLimit:         "",                   // Not provided
			GasPrice:         "",                   // Not provided
			Nonce:            "",                   // Not provided
			Type:             types.TxTypeTransfer, // Standard token transfer
			CoinType:         types.CoinTypeToken,  // Token type
			TokenDisplayName: tt.Token.Symbol,
			Decimals:         decimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          tt.Token.IconURL,
		}

		transactions = append(transactions, transaction)
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Transformed token transfers from Blockscout")

	return transactions
}
