package ankr

import (
	"strings"
	"tx-aggregator/config"
	"tx-aggregator/internal/chainmeta"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"tx-aggregator/utils"
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
func (a *AnkrProvider) transformAnkrTokenTransfers(
	resp *model.AnkrTokenTransferResponse,
	address string,
) []model.Transaction {
	if resp == nil || resp.Result.Transfers == nil {
		logger.Log.Warn().Msg("No token transfers to transform")
		return nil
	}

	logger.Log.Debug().
		Int("transfer_count", len(resp.Result.Transfers)).
		Msg("Transforming token transfers")

	var transactions []model.Transaction

	for _, tr := range resp.Result.Transfers {
		chainID, err := chainmeta.AnkrChainIDByName(tr.Blockchain)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("blockchain", tr.Blockchain).
				Msg("Failed to get chain ID from Ankr")
		}

		// Determine transaction direction
		tranType := model.TransTypeOut
		if strings.EqualFold(tr.ToAddress, address) {
			tranType = model.TransTypeIn
		}

		balance, err := utils.MultiplyByDecimals(tr.Value, int(tr.TokenDecimals))
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("address", address).
				Msg("Failed to normalize token transfer amount")
		}

		// Construct transaction object
		transaction := model.Transaction{
			ChainID:          chainID,
			TokenID:          0,
			State:            model.TxStateSuccess, // always mark as success (API limitation)
			Height:           tr.BlockHeight,
			Hash:             tr.TransactionHash,
			BlockHash:        "", // not available from API
			FromAddress:      tr.FromAddress,
			ToAddress:        tr.ToAddress,
			TokenAddress:     tr.ContractAddress,
			Balance:          balance,
			Amount:           tr.Value,
			GasUsed:          "", // not provided
			GasLimit:         "", // not available
			GasPrice:         "", // not available
			Nonce:            "", // not available
			Type:             model.TxTypeTransfer,
			CoinType:         model.CoinTypeToken,
			TokenDisplayName: tr.TokenName,
			Decimals:         tr.TokenDecimals,
			CreatedTime:      tr.Timestamp,
			ModifiedTime:     tr.Timestamp,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          tr.Thumbnail,
		}

		transactions = append(transactions, transaction)
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Successfully transformed token transfers")

	return transactions
}
