package provider

import (
	"strings"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// GetTransactionsByAddress retrieves normal transactions from Ankr for the given address
// These are native token transfers (ETH, BNB, MATIC, etc.)
func (p *AnkrProvider) GetTransactionsByAddress(address string) (*model.AnkrTransactionResponse, error) {
	logger.Log.Debug().
		Str("address", address).
		Msg("Fetching normal transactions from Ankr")

	requestBody := model.AnkrTransactionRequest{
		JSONRPC: "2.0",
		Method:  "ankr_getTransactionsByAddress",
		Params: map[string]interface{}{
			"blockchain":  config.AppConfig.Ankr.RequestBlockchains,
			"includeLogs": true,
			"descOrder":   true,
			"pageSize":    config.AppConfig.Ankr.RequestPageSize,
			"address":     address,
		},
		ID: 1,
	}

	var result model.AnkrTransactionResponse
	if err := p.sendRequest(requestBody, &result); err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch normal transactions from Ankr")
		return nil, err
	}

	logger.Log.Debug().
		Str("address", address).
		Int("tx_count", len(result.Result.Transactions)).
		Msg("Successfully fetched normal transactions")
	return &result, nil
}

// transformAnkrNormalTx converts AnkrTransactionResponse into a slice of model.Transaction
// These are native token transfers (ETH, BNB, MATIC, etc.)
func (a *AnkrProvider) transformAnkrNormalTx(resp *model.AnkrTransactionResponse, address string) []model.Transaction {
	if resp == nil || resp.Result.Transactions == nil {
		logger.Log.Warn().Msg("No normal transactions to transform")
		return nil
	}

	logger.Log.Debug().
		Int("tx_count", len(resp.Result.Transactions)).
		Msg("Transforming normal transactions")

	var transactions []model.Transaction

	for _, tx := range resp.Result.Transactions {
		chainID, _ := config.AnkrChainIDByName(tx.Blockchain)
		height := parseStringToInt64OrDefault(tx.BlockNumber, 0)
		timestamp := parseStringToInt64OrDefault(tx.Timestamp, 0)
		txIndex := parseStringToInt64OrDefault(tx.TransactionIndex, 0)

		// Determine transaction state
		var state int
		if parseStringToInt64OrDefault(tx.Status, 0) == model.TxStateSuccess {
			state = model.TxStateSuccess
		} else {
			state = model.TxStateFail
		}

		// Normalize values
		amount, _ := NormalizeNumericString(tx.Value)
		gasLimit, _ := NormalizeNumericString(tx.Gas)
		gasUsed, _ := NormalizeNumericString(tx.GasUsed)
		gasPrice, _ := NormalizeNumericString(tx.GasPrice)
		nonce, _ := NormalizeNumericString(tx.Nonce)

		// Detect ERC20 type and approve value
		txType, tokenAddr, approveValue := DetectERC20TypeForAnkr(tx.Logs)
		approveShow := ""
		if txType == model.TxTypeApprove {
			approveShow = approveValue
		}

		// Determine transaction direction
		tranType := model.TransTypeOut
		if strings.EqualFold(tx.To, address) {
			tranType = model.TransTypeIn
		}

		// Build transaction model
		transaction := model.Transaction{
			ChainID:          chainID,
			TokenID:          0,
			State:            state,
			Height:           height,
			Hash:             tx.Hash,
			TxIndex:          txIndex,
			BlockHash:        tx.BlockHash,
			FromAddress:      tx.From,
			ToAddress:        tx.To,
			TokenAddress:     tokenAddr,
			Amount:           amount,
			GasUsed:          gasUsed,
			GasLimit:         gasLimit,
			GasPrice:         gasPrice,
			Nonce:            nonce,
			Type:             txType,
			CoinType:         model.CoinTypeNative,
			TokenDisplayName: "",
			Decimals:         model.NativeDefaultDecimals,
			CreatedTime:      timestamp,
			ModifiedTime:     timestamp,
			TranType:         tranType,
			ApproveShow:      approveShow,
			IconURL:          "",
		}

		transactions = append(transactions, transaction)
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Successfully transformed normal transactions")

	return transactions
}
