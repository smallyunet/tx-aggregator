package blockscout

import (
	"fmt"
	"strconv"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// fetchBlockscoutNormalTx retrieves normal transactions from the Blockscout endpoint:
// GET /addresses/{address}/transactions
func (t *BlockscoutProvider) fetchBlockscoutNormalTx(address string) (*types.BlockscoutTransactionResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/transactions?limit=%d", t.config.URL, address, t.config.RequestPageSize)
	var result types.BlockscoutTransactionResponse
	if err := utils.DoHttpRequestWithLogging("GET", "blockscout.normalTx", url, nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// transformBlockscoutNormalTx is the initial conversion of Blockscout normal transactions response to []model.Transaction.
// This function does NOT perform ERC20 approve detection—only base transaction fields are handled.
func (t *BlockscoutProvider) transformBlockscoutNormalTx(
	resp *types.BlockscoutTransactionResponse,
	address string,
	logsMap map[string][]types.BlockscoutLog, // May be nil on first pass
) []types.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No normal transactions to transform from Blockscout")
		return nil
	}

	var transactions []types.Transaction

	for _, tx := range resp.Items {
		// Determine transaction status
		state := types.TxStateFail
		if strings.EqualFold(tx.Status, "ok") {
			state = types.TxStateSuccess
		}

		// Determine transaction direction
		tranType := types.TransTypeOut
		if strings.EqualFold(tx.To.Hash, address) {
			tranType = types.TransTypeIn
		}

		// Parse timestamp
		unixTime := utils.ParseBlockscoutTimestampToUnix(tx.Timestamp)

		// Normalize values
		amountRaw, err := utils.NormalizeNumericString(tx.Value)
		amount := utils.DivideByDecimals(amountRaw, types.NativeDefaultDecimals)
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

		nativeTokenName, err := utils.NativeTokenByChainID(t.chainID)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Int64("chain_id", t.chainID).
				Msg("Failed to get native token name")
		}

		// Construct the transaction
		transaction := types.Transaction{
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
			Type:             types.TxTypeUnknown,  // Default type for native transfer
			CoinType:         types.CoinTypeNative, // Native coin
			TokenDisplayName: nativeTokenName,
			Decimals:         types.NativeDefaultDecimals,
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
	normalTxs []types.Transaction,
	logsMap map[string][]types.BlockscoutLog,
	address string,
) []types.Transaction {

	for i, tx := range normalTxs {
		// Does this transaction have logs in the map?
		logsForTx, found := logsMap[tx.Hash]
		if !found || len(logsForTx) == 0 {
			// No logs => nothing to detect
			continue
		}

		// We’ll see if any log indicates a recognized ERC-20 event
		var finalTxType int = types.TxTypeUnknown
		var finalTokenAddr, finalApproveVal string

		for _, lg := range logsForTx {
			// Our generic detection function:
			// DetectERC20Event(contractAddress, topics, data)
			txType, tokenAddr, approveValue := utils.DetectERC20Event(
				lg.Address.Hash, // the contract address
				lg.Topics,
				lg.Data,
			)

			if txType != types.TxTypeUnknown {
				finalTxType = txType
				finalTokenAddr = tokenAddr
				finalApproveVal = approveValue

				// If you only want to capture the first recognized event,
				// you can break here.
				break
			}
		}

		// If we recognized an ERC-20 event, update the transaction
		if finalTxType != types.TxTypeUnknown {
			// If it’s an Approval, store the approval value
			if finalTxType == types.TxTypeApprove {
				normalTxs[i].ApproveShow = finalApproveVal
			}

			normalTxs[i].Type = finalTxType
			normalTxs[i].TokenAddress = finalTokenAddr
		}
	}

	return normalTxs
}
