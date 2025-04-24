package quicknode

import (
	"strconv"
	"strings"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

// -------------------------- fetch & transform ----------------------------

func (q *QuickNodeProvider) getWalletTokenTransfers(addr, contract string, page, perPage int) (*quickNodeTokenResp, error) {
	param := map[string]interface{}{
		"address": addr,
		"page":    page,
		"perPage": perPage,
	}
	if contract != "" {
		param["contract"] = contract
	}

	req := quickNodeTokenReq{
		JSONRPC: "2.0",
		Method:  "qn_getWalletTokenTransactions",
		Params:  []interface{}{param},
		ID:      1,
	}

	var resp quickNodeTokenResp
	if err := q.sendRequest(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (q *QuickNodeProvider) transformQuickNodeToken(resp *quickNodeTokenResp, addr string) []types.Transaction {
	if resp == nil || len(resp.Result.Transfers) == 0 {
		return nil
	}

	decimals64, _ := strconv.ParseInt(resp.Result.Token.Decimals, 10, 64)
	tokenName := resp.Result.Token.Name
	tokenAddr := resp.Result.Token.ContractAddress

	var out []types.Transaction
	for _, tr := range resp.Result.Transfers {
		height := utils.ParseStringToInt64OrDefault(tr.BlockNumber, 0)
		timestamp := utils.ParseStringToInt64OrDefault(tr.Timestamp, 0)

		var (
			rawValue string
			tranType = types.TransTypeOut
		)

		if strings.EqualFold(tr.FromAddress, addr) {
			rawValue = tr.SentAmount
			tranType = types.TransTypeOut
		} else {
			rawValue = tr.ReceivedAmount
			tranType = types.TransTypeIn
		}

		amount := utils.DivideByDecimals(rawValue, int(decimals64))

		out = append(out, types.Transaction{
			ChainID:          q.chainID,
			TokenID:          0,
			State:            types.TxStateSuccess,
			Height:           height,
			Hash:             tr.TransactionHash,
			BlockHash:        "",
			FromAddress:      tr.FromAddress,
			ToAddress:        tr.ToAddress,
			TokenAddress:     tokenAddr,
			Balance:          rawValue,
			Amount:           amount,
			GasUsed:          "",
			GasLimit:         "",
			GasPrice:         "",
			Nonce:            "",
			Type:             types.TxTypeTransfer,
			CoinType:         types.CoinTypeToken,
			TokenDisplayName: tokenName,
			Decimals:         decimals64,
			CreatedTime:      timestamp,
			ModifiedTime:     timestamp,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          "",
		})
	}
	return out
}
