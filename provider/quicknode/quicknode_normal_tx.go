package quicknode

import (
	"strings"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// ---------------------------- fetch & transform ---------------------------

func (q *QuickNodeProvider) getTxByAddress(addr string, page, perPage int) (*quickNodeTxResponse, error) {
	req := quickNodeTxRequest{
		JSONRPC: "2.0",
		Method:  "qn_getTransactionsByAddress",
		Params: []interface{}{
			map[string]interface{}{
				"address": addr,
				"page":    page,
				"perPage": perPage,
			},
		},
		ID: 1,
	}

	var resp quickNodeTxResponse
	if err := q.sendRequest(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (q *QuickNodeProvider) transformQuickNodeNative(resp *quickNodeTxResponse, addr string) []types.Transaction {
	if resp == nil || len(resp.Result.Transactions) == 0 {
		return nil
	}

	var out []types.Transaction
	for _, tx := range resp.Result.Transactions {
		height := utils.ParseStringToInt64OrDefault(tx.BlockNumber, 0)
		timestamp := utils.ParseStringToInt64OrDefault(tx.BlockTimestamp, 0)
		index := utils.ParseStringToInt64OrDefault(tx.TransactionIndex, 0)

		rawValue, _ := utils.NormalizeNumericString(tx.Value)
		amount := utils.DivideByDecimals(rawValue, types.NativeDefaultDecimals)

		tranType := types.TransTypeOut
		if strings.EqualFold(tx.ToAddress, addr) {
			tranType = types.TransTypeIn
		}

		state := types.TxStateFail
		if strings.HasPrefix(tx.Status, "0x1") || tx.Status == "1" {
			state = types.TxStateSuccess
		}

		out = append(out, types.Transaction{
			ChainID:          q.chainID,
			TokenID:          0,
			State:            state,
			Height:           height,
			Hash:             tx.TransactionHash,
			TxIndex:          index,
			BlockHash:        "", // not supplied
			FromAddress:      tx.FromAddress,
			ToAddress:        tx.ToAddress,
			TokenAddress:     tx.ContractAddress,
			Balance:          rawValue,
			Amount:           amount,
			GasUsed:          "",
			GasLimit:         "",
			GasPrice:         "",
			Nonce:            "",
			Type:             types.TxTypeTransfer,
			CoinType:         types.CoinTypeNative,
			TokenDisplayName: "",
			Decimals:         types.NativeDefaultDecimals,
			CreatedTime:      timestamp,
			ModifiedTime:     timestamp,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          "",
		})
	}
	return out
}
