package ankr

import (
	"strconv"
	"strings"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// GetTokenTransfers retrieves token transfer events from Ankr for the given address
// These are ERC20/BEP20/etc token transfers
func (p *AnkrProvider) GetTokenTransfers(params *types.TransactionQueryParams) (*types.AnkrTokenTransferResponse, error) {
	address := params.Address

	// Resolve chain list for this request
	blockchains, err := utils.ResolveAnkrBlockchains(params.ChainNames)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Strs("params_chainNames", params.ChainNames).
			Msg("invalid chainNames parameter")
		return nil, err
	}

	logger.Log.Debug().
		Str("address", address).
		Strs("ankr_chainNames", blockchains).
		Str("include_logs", strconv.FormatBool(config.Current().Ankr.IncludeLogs)).
		Str("desc_order", strconv.FormatBool(config.Current().Ankr.DescOrder)).
		Int("page_size", config.Current().Ankr.RequestPageSize).
		Msg("Fetching token transfers from Ankr")

	requestBody := types.AnkrTransactionRequest{
		JSONRPC: "2.0",
		Method:  "ankr_getTokenTransfers",
		Params: map[string]interface{}{
			"blockchain": blockchains,
			"descOrder":  config.Current().Ankr.DescOrder,
			"pageSize":   config.Current().Ankr.RequestPageSize,
			"address":    address,
		},
		ID: 1,
	}

	var result types.AnkrTokenTransferResponse
	if err := p.sendRequest(requestBody, &result, "tokenTx"); err != nil {
		logger.Log.Error().
			Err(err).
			Str("address", address).
			Msg("Failed to fetch token transfers from Ankr")
		return nil, err
	}

	if result.Error != nil {
		logger.Log.Error().
			Int("error_code", result.Error.Code).
			Str("error_message", result.Error.Message).
			Str("address", address).
			Msg("Ankr API returned an error in token transfer response")
		return nil, result.Error // OK now, since it implements error
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
	resp *types.AnkrTokenTransferResponse,
	address string,
) []types.Transaction {
	if resp == nil || resp.Result.Transfers == nil {
		logger.Log.Warn().Msg("No token transfers to transform")
		return nil
	}

	logger.Log.Debug().
		Int("transfer_count", len(resp.Result.Transfers)).
		Msg("Transforming token transfers")

	var transactions []types.Transaction

	for _, tr := range resp.Result.Transfers {
		chainID, err := utils.AnkrChainIDByName(tr.Blockchain)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("blockchain", tr.Blockchain).
				Msg("Failed to get chain ID from Ankr")
		}

		// Determine transaction direction
		tranType := types.TransTypeOut
		if strings.EqualFold(tr.ToAddress, address) {
			tranType = types.TransTypeIn
		}

		balance, err := utils.MultiplyByDecimals(tr.Value, int(tr.TokenDecimals))
		if err != nil {
			logger.Log.Error().
				Err(err).
				Str("address", address).
				Msg("Failed to normalize token transfer amount")
		}

		// Construct transaction object
		transaction := types.Transaction{
			ChainID:          chainID,
			TokenID:          0,
			State:            types.TxStateSuccess, // always mark as success (API limitation)
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
			Type:             types.TxTypeTransfer,
			CoinType:         types.CoinTypeToken,
			TokenDisplayName: tr.TokenSymbol,
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
