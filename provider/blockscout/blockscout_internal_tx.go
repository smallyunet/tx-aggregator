package blockscout

import (
	"fmt"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"tx-aggregator/utils"
)

// fetchBlockscoutInternalTx retrieves internal transactions from Blockscout:
// GET /addresses/{address}/internal-transactions
func (t *BlockscoutProvider) fetchBlockscoutInternalTx(address string) (*model.BlockscoutInternalTxResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/internal-transactions?limit=%d", t.config.URL, address, t.config.RequestPageSize)
	var result model.BlockscoutInternalTxResponse
	if err := utils.DoHttpRequestWithLogging("GET", "blockscout.internalTx", url, nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// transformBlockscoutInternalTx converts internal transaction data into []model.Transaction.
// This version captures basic transfer data without logs or deep inspection.
func (t *BlockscoutProvider) transformBlockscoutInternalTx(
	resp *model.BlockscoutInternalTxResponse,
	address string,
) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No internal transactions to transform from Blockscout")
		return nil
	}

	var transactions []model.Transaction

	for _, itx := range resp.Items {
		// Determine transaction success state
		state := model.TxStateFail
		if itx.Success {
			state = model.TxStateSuccess
		}

		// Parse timestamp to Unix time
		unixTime := utils.ParseBlockscoutTimestampToUnix(itx.Timestamp)

		// Safely extract from/to addresses
		fromHash := ""
		toHash := ""
		if itx.From != nil {
			fromHash = itx.From.Hash
		}
		if itx.To != nil {
			toHash = itx.To.Hash
		}

		// Determine transaction direction
		tranType := model.TransTypeOut
		if strings.EqualFold(toHash, address) {
			tranType = model.TransTypeIn
		}

		// Normalize gas limit (if provided)
		gasLimit, err := utils.NormalizeNumericString(itx.GasLimit)
		amountRaw, err := utils.NormalizeNumericString(itx.Value)
		amount := utils.DivideByDecimals(amountRaw, model.NativeDefaultDecimals)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("address", address).
				Msg("Failed to normalize internal transaction amount")
		}

		// Construct transaction object
		transaction := model.Transaction{
			ChainID:          t.chainID,
			TokenID:          0,
			State:            state,
			Height:           itx.BlockNumber,
			Hash:             itx.TransactionHash, // Uses outer transaction hash
			BlockHash:        "",                  // Not available for internal tx
			FromAddress:      fromHash,
			ToAddress:        toHash,
			TokenAddress:     "",
			Balance:          amountRaw,
			Amount:           amount,
			GasUsed:          "", // Not provided
			GasLimit:         gasLimit,
			GasPrice:         "",
			Nonce:            "",
			Type:             model.TxTypeInternal, // Internal call
			CoinType:         model.CoinTypeNative, // Typically native token
			TokenDisplayName: "",
			Decimals:         model.NativeDefaultDecimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          "",
		}

		transactions = append(transactions, transaction)
	}

	return transactions
}
