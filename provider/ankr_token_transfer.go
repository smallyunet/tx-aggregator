package provider

import (
	"strings"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// GetTokenTransfers retrieves token transfer events from Ankr for the given address
// These are ERC20/BEP20/etc token transfers
func (p *AnkrProvider) GetTokenTransfers(address string) (*model.AnkrTokenTransferResponse, error) {
	logger.Log.Debug().
		Str("address", address).
		Msg("Fetching token transfers from Ankr")

	requestBody := model.AnkrTransactionRequest{
		JSONRPC: "2.0",
		Method:  "ankr_getTokenTransfers",
		Params: map[string]interface{}{
			"blockchain": config.AppConfig.Ankr.RequestBlockchains,
			"pageSize":   config.AppConfig.Ankr.RequestPageSize,
			"address":    address,
		},
		ID: 1,
	}

	var result model.AnkrTokenTransferResponse
	if err := p.sendRequest(requestBody, &result); err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch token transfers from Ankr")
		return nil, err
	}

	logger.Log.Debug().
		Str("address", address).
		Int("transfer_count", len(result.Result.Transfers)).
		Msg("Successfully fetched token transfers")
	return &result, nil
}

// transformAnkrTokenTransfers converts AnkrTokenTransferResponse into a slice of model.Transaction
// These represent ERC20/BEP20/etc token transfers
func (a *AnkrProvider) transformAnkrTokenTransfers(resp *model.AnkrTokenTransferResponse, address string) []model.Transaction {
	if resp == nil || resp.Result.Transfers == nil {
		logger.Log.Warn().
			Msg("No token transfers to transform")
		return nil
	}

	logger.Log.Debug().
		Int("transfer_count", len(resp.Result.Transfers)).
		Msg("Transforming token transfers")

	var transactions []model.Transaction
	for _, tr := range resp.Result.Transfers {
		chainID, _ := config.ChainIDByName(tr.Blockchain)

		tranType := model.TransTypeOut // default to outgoing
		if strings.EqualFold(tr.ToAddress, address) {
			tranType = model.TransTypeIn
		}

		transactions = append(transactions, model.Transaction{
			ChainID:          chainID,
			TokenID:          0,
			State:            1,
			Height:           int64(tr.BlockHeight),
			Hash:             tr.TransactionHash,
			BlockHash:        "", // not provided by transfer API
			FromAddress:      tr.FromAddress,
			ToAddress:        tr.ToAddress,
			TokenAddress:     tr.ContractAddress,
			Amount:           tr.Value,
			GasUsed:          "",                   // not provided by transfer API
			GasLimit:         "",                   // not available
			GasPrice:         "",                   // not available
			Nonce:            "",                   // not available
			Type:             model.TxTypeTransfer, // default to transfer
			CoinType:         model.CoinTypeToken,  // 2 = token
			TokenDisplayName: tr.TokenName,
			Decimals:         tr.TokenDecimals,
			CreatedTime:      tr.Timestamp,
			ModifiedTime:     tr.Timestamp,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          tr.Thumbnail, // optional logo/image
		})
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Successfully transformed token transfers")
	return transactions
}
