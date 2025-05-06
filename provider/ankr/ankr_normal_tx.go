package ankr

import (
	"strings"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// GetTransactionsByAddress retrieves normal transactions from Ankr for the given address
// These are native token transfers (ETH, BNB, MATIC, etc.)
func (p *AnkrProvider) GetTransactionsByAddress(params *types.TransactionQueryParams) (*types.AnkrTransactionResponse, error) {
	address := params.Address

	// Resolve chain list for this request
	blockchains, err := utils.ResolveAnkrBlockchains(params.ChainNames)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("invalid chainNames parameter")
		return nil, err
	}

	logger.Log.Debug().
		Str("address", address).
		Msg("Fetching normal transactions from Ankr")

	requestBody := types.AnkrTransactionRequest{
		JSONRPC: "2.0",
		Method:  "ankr_getTransactionsByAddress",
		Params: map[string]interface{}{
			"blockchain":  blockchains,
			"includeLogs": true,
			"descOrder":   true,
			"pageSize":    config.Current().Ankr.RequestPageSize,
			"address":     address,
		},
		ID: 1,
	}

	var result types.AnkrTransactionResponse
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
func (a *AnkrProvider) transformAnkrNormalTx(resp *types.AnkrTransactionResponse, address string) []types.Transaction {
	if resp == nil || resp.Result.Transactions == nil {
		logger.Log.Warn().Msg("No normal transactions to transform")
		return nil
	}

	logger.Log.Debug().
		Int("tx_count", len(resp.Result.Transactions)).
		Msg("Transforming normal transactions")

	var transactions []types.Transaction

	for _, tx := range resp.Result.Transactions {
		chainID, _ := utils.AnkrChainIDByName(tx.Blockchain)
		height := utils.ParseStringToInt64OrDefault(tx.BlockNumber, 0)
		timestamp := utils.ParseStringToInt64OrDefault(tx.Timestamp, 0)
		txIndex := utils.ParseStringToInt64OrDefault(tx.TransactionIndex, 0)

		// Determine transaction state
		var state int
		if utils.ParseStringToInt64OrDefault(tx.Status, 0) == types.TxStateSuccess {
			state = types.TxStateSuccess
		} else {
			state = types.TxStateFail
		}

		// Normalize values
		amountRaw, err := utils.NormalizeNumericString(tx.Value)
		amount := utils.DivideByDecimals(amountRaw, types.NativeDefaultDecimals)
		gasLimit, err := utils.NormalizeNumericString(tx.Gas)
		gasUsed, err := utils.NormalizeNumericString(tx.GasUsed)
		gasPrice, err := utils.NormalizeNumericString(tx.GasPrice)
		nonce, err := utils.NormalizeNumericString(tx.Nonce)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("address", address).
				Msg("Failed to normalize transaction values")
		}

		// Detect ERC20 type and approve value
		txType, tokenAddr, approveValue := utils.DetectERC20TypeForAnkr(tx.Logs)
		approveShow := ""
		if txType == types.TxTypeApprove {
			approveShow = approveValue
		}

		// Determine transaction direction
		tranType := types.TransTypeOut
		if strings.EqualFold(tx.To, address) {
			tranType = types.TransTypeIn
		}

		// Build transaction types
		transaction := types.Transaction{
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
			Balance:          amountRaw,
			Amount:           amount,
			GasUsed:          gasUsed,
			GasLimit:         gasLimit,
			GasPrice:         gasPrice,
			Nonce:            nonce,
			Type:             txType,
			CoinType:         types.CoinTypeNative,
			TokenDisplayName: "",
			Decimals:         types.NativeDefaultDecimals,
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
