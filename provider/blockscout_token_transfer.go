package provider

import (
	"fmt"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// fetchBlockscoutTokenTransfers retrieves token transfers from Blockscout:
// GET /addresses/{address}/token-transfers
func (t *BlockscoutProvider) fetchBlockscoutTokenTransfers(address string) (*model.BlockscoutTokenTransferResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/token-transfers?limit=%d", t.config.URL, address, t.config.RequestPageSize)
	var result model.BlockscoutTokenTransferResponse
	if err := DoHttpRequestWithLogging("GET", "blockscout.tokenTransfers", url, nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// / transformBlockscoutTokenTransfers converts Blockscout token transfers into []model.Transaction.
func (t *BlockscoutProvider) transformBlockscoutTokenTransfers(
	resp *model.BlockscoutTokenTransferResponse,
	address string,
) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No token transfers to transform from Blockscout")
		return nil
	}

	var transactions []model.Transaction

	for _, tt := range resp.Items {
		// Determine transaction direction
		tranType := model.TransTypeOut
		if strings.EqualFold(tt.To.Hash, address) {
			tranType = model.TransTypeIn
		}

		// Parse timestamp and decimals
		unixTime := parseBlockscoutTimestampToUnix(tt.Timestamp)
		decimals := parseStringToInt64OrDefault(tt.Token.Decimals, 18) // Default to 18 if missing

		// Build transaction object
		transaction := model.Transaction{
			ChainID:          t.chainID,
			TokenID:          0,
			State:            model.TxStateSuccess, // Token transfers are assumed successful
			Height:           tt.BlockNumber,
			Hash:             tt.TransactionHash,
			BlockHash:        tt.BlockHash,
			FromAddress:      tt.From.Hash,
			ToAddress:        tt.To.Hash,
			TokenAddress:     tt.Token.Address,
			Amount:           tt.Total.Value,
			GasUsed:          "",                   // Not provided
			GasLimit:         "",                   // Not provided
			GasPrice:         "",                   // Not provided
			Nonce:            "",                   // Not provided
			Type:             model.TxTypeTransfer, // Standard token transfer
			CoinType:         model.CoinTypeToken,  // Token type
			TokenDisplayName: tt.Token.Name,
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
