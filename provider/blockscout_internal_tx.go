package provider

import (
	"fmt"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// fetchBlockscoutInternalTx retrieves internal transactions from Blockscout:
// GET /addresses/{address}/internal-transactions
func (t *BlockscoutProvider) fetchBlockscoutInternalTx(address string) (*model.BlockscoutInternalTxResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/internal-transactions?limit=%d", t.config.URL, address, t.config.RequestPageSize)
	var result model.BlockscoutInternalTxResponse
	if err := DoHttpRequestWithLogging("GET", "blockscout.internalTx", url, nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// transformBlockscoutInternalTx converts internal transaction data into []model.Transaction.
// If you only want to store minimal details, this is an example approach.
func (t *BlockscoutProvider) transformBlockscoutInternalTx(resp *model.BlockscoutInternalTxResponse, address string) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No internal transactions to transform from Blockscout")
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

		gasLimit, _ := NormalizeNumericString(itx.GasLimit)

		transactions = append(transactions, model.Transaction{
			ChainID:          t.chainID,
			State:            state,
			Height:           itx.BlockNumber,
			Hash:             itx.TransactionHash, // internal tx uses outer transaction's hash
			BlockHash:        "",                  // not provided by Blockscout for internal
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
