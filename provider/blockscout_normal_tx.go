package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// fetchBlockscoutNormalTx retrieves normal transactions from the Tantin endpoint:
// GET /addresses/{address}/transactions
func (t *BlockscoutProvider) fetchBlockscoutNormalTx(address string) (*model.BlockscoutTransactionResponse, error) {
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

	var result model.BlockscoutTransactionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal normal transactions: %w", err)
	}

	return &result, nil
}

// transformBlockscoutNormalTx is the initial conversion of Tantin normal transactions response to []model.Transaction.
// Here we do NOT yet handle "approve" detection. We simply store the base transaction data.
func (t *BlockscoutProvider) transformBlockscoutNormalTx(
	resp *model.BlockscoutTransactionResponse,
	address string,
	logsMap map[string][]model.BlockscoutLog, // May be nil on first pass
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
