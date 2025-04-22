package blockscout

import (
	"fmt"
	"strconv"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"tx-aggregator/utils"
)

// fetchBlockscoutNormalTx retrieves normal transactions from the Blockscout endpoint:
// GET /addresses/{address}/transactions
func (t *BlockscoutProvider) fetchBlockscoutNormalTx(address string) (*model.BlockscoutTransactionResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/transactions?limit=%d", t.config.URL, address, t.config.RequestPageSize)
	var result model.BlockscoutTransactionResponse
	if err := utils.DoHttpRequestWithLogging("GET", "blockscout.normalTx", url, nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// transformBlockscoutNormalTx is the initial conversion of Blockscout normal transactions response to []model.Transaction.
// This function does NOT perform ERC20 approve detection—only base transaction fields are handled.
func (t *BlockscoutProvider) transformBlockscoutNormalTx(
	resp *model.BlockscoutTransactionResponse,
	address string,
	logsMap map[string][]model.BlockscoutLog, // May be nil on first pass
) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No normal transactions to transform from Blockscout")
		return nil
	}

	var transactions []model.Transaction

	for _, tx := range resp.Items {
		// Determine transaction status
		state := model.TxStateFail
		if strings.EqualFold(tx.Status, "ok") {
			state = model.TxStateSuccess
		}

		// Determine transaction direction
		tranType := model.TransTypeOut
		if strings.EqualFold(tx.To.Hash, address) {
			tranType = model.TransTypeIn
		}

		// Parse timestamp
		unixTime := utils.ParseBlockscoutTimestampToUnix(tx.Timestamp)

		// Normalize values
		amountRaw, err := utils.NormalizeNumericString(tx.Value)
		amount := utils.DivideByDecimals(amountRaw, model.NativeDefaultDecimals)
		gasUsed, err := utils.NormalizeNumericString(tx.GasUsed)
		gasLimit, err := utils.NormalizeNumericString(tx.GasLimit)
		gasPrice, err := utils.NormalizeNumericString(tx.GasPrice)
		nonce, err := utils.NormalizeNumericString(strconv.FormatInt(tx.Nonce, 10))
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("address", address).
				Msg("Failed to normalize transaction nonce")
		}

		// Construct the transaction
		transaction := model.Transaction{
			ChainID:          t.chainID,
			TokenID:          0,
			State:            state,
			Height:           tx.BlockNumber,
			Hash:             tx.Hash,
			BlockHash:        tx.BlockHash,
			FromAddress:      tx.From.Hash,
			ToAddress:        tx.To.Hash,
			TokenAddress:     "",
			Balance:          amountRaw,
			Amount:           amount,
			GasUsed:          gasUsed,
			GasLimit:         gasLimit,
			GasPrice:         gasPrice,
			Nonce:            nonce,
			Type:             model.TxTypeUnknown,  // Default type for native transfer
			CoinType:         model.CoinTypeNative, // Native coin
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

// transformBlockscoutNormalTxWithLogs re-processes the already converted normal transactions
// to detect if any are "approve" type (or other ERC-20 events) by scanning the logs map.
// `logsMap` is keyed by tx hash => slice of BlockscoutLog.
func (b *BlockscoutProvider) transformBlockscoutNormalTxWithLogs(
	normalTxs []model.Transaction,
	logsMap map[string][]model.BlockscoutLog,
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
			txType, tokenAddr, approveValue := utils.DetectERC20Event(
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
