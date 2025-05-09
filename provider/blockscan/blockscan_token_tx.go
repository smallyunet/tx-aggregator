package blockscan

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

func (p *BlockscanProvider) fetchTokenTx(addr string) (*types.BlockscanTokenTxResp, error) {
	q := url.Values{
		"module":  {"account"},
		"action":  {"tokentx"},
		"address": {addr},
		"page":    {strconv.FormatInt(p.cfg.Page, 10)},
		"offset":  {fmt.Sprint(p.cfg.RequestPageSize)},
		"sort":    {p.cfg.Sort},
		"apikey":  {p.cfg.APIKey},
	}
	var out types.BlockscanTokenTxResp
	u := fmt.Sprintf("%s?%s", p.cfg.URL, q.Encode())
	if err := utils.DoHttpRequestWithLogging("GET", "blockscan.tokenTx", u, nil, nil, &out); err != nil {
		return nil, err
	}

	if out.Status == types.StatusError {
		logger.Log.Warn().
			Str("error_message", out.Message).
			Str("address", addr).
			Msg("Failed to fetch token transactions from Blockscan")
		return nil, fmt.Errorf("blockscan error: %s", out.Message)
	}

	return &out, nil
}

func (p *BlockscanProvider) transformTokenTx(resp *types.BlockscanTokenTxResp, addr string) []types.Transaction {
	if resp == nil || resp.Status != types.StatusOK || len(resp.Result) == 0 {
		return nil
	}

	var txs []types.Transaction
	for _, tt := range resp.Result {
		height := utils.ParseStringToInt64OrDefault(tt.BlockNumber, 0)
		unixTime := utils.ParseStringToInt64OrDefault(tt.TimeStamp, 0)
		txIndex := utils.ParseStringToInt64OrDefault(tt.TransactionIndex, 0)
		decimals := utils.ParseStringToInt64OrDefault(tt.TokenDecimal, types.NativeDefaultDecimals)

		balanceRaw, _ := utils.NormalizeNumericString(tt.Value)
		amount := utils.DivideByDecimals(balanceRaw, int(decimals))

		tranType := types.TransTypeOut
		if strings.EqualFold(tt.To, addr) {
			tranType = types.TransTypeIn
		}

		gasLimit, _ := utils.NormalizeNumericString(tt.Gas)
		gasUsed, _ := utils.NormalizeNumericString(tt.GasUsed)
		gasPrice, _ := utils.NormalizeNumericString(tt.GasPrice)

		txs = append(txs, types.Transaction{
			ChainID:          p.chainID,
			Height:           height,
			Hash:             tt.Hash,
			BlockHash:        tt.BlockHash,
			TxIndex:          txIndex,
			FromAddress:      tt.From,
			ToAddress:        tt.To,
			TokenAddress:     tt.ContractAddress,
			Balance:          balanceRaw,
			Amount:           amount,
			GasLimit:         gasLimit,
			GasUsed:          gasUsed,
			GasPrice:         gasPrice,
			Type:             types.TxTypeTransfer,
			CoinType:         types.CoinTypeToken,
			TokenDisplayName: tt.TokenSymbol,
			Decimals:         decimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
		})
	}
	return txs
}
