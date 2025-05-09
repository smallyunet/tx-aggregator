package blockscan

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"tx-aggregator/types"
	"tx-aggregator/utils"
)

func (p *BlockscanProvider) fetchInternalTx(addr string) (*types.BlockscanInternalTxResp, error) {
	q := url.Values{
		"module":     {"account"},
		"action":     {"txlistinternal"},
		"address":    {addr},
		"startblock": {strconv.FormatInt(p.cfg.Startblock, 10)},
		"endblock":   {strconv.FormatInt(p.cfg.Endblock, 10)},
		"page":       {strconv.FormatInt(p.cfg.Page, 10)},
		"offset":     {fmt.Sprint(p.cfg.RequestPageSize)},
		"sort":       {p.cfg.Sort},
		"apikey":     {p.cfg.APIKey},
	}
	var out types.BlockscanInternalTxResp
	u := fmt.Sprintf("%s?%s", p.cfg.URL, q.Encode())
	if err := utils.DoHttpRequestWithLogging("GET", "blockscan.internalTx", u, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *BlockscanProvider) transformInternalTx(resp *types.BlockscanInternalTxResp, addr string) []types.Transaction {
	if resp == nil || resp.Status != "1" || len(resp.Result) == 0 {
		return nil
	}

	var txs []types.Transaction
	for _, it := range resp.Result {
		height := utils.ParseStringToInt64OrDefault(it.BlockNumber, 0)
		unixTime := utils.ParseStringToInt64OrDefault(it.TimeStamp, 0)

		state := types.TxStateFail
		if it.IsError == "0" {
			state = types.TxStateSuccess
		}

		tranType := types.TransTypeOut
		if strings.EqualFold(it.To, addr) {
			tranType = types.TransTypeIn
		}

		valueRaw, _ := utils.NormalizeNumericString(it.Value)
		value := utils.DivideByDecimals(valueRaw, types.NativeDefaultDecimals)
		gasLimit, _ := utils.NormalizeNumericString(it.Gas)
		gasUsed, _ := utils.NormalizeNumericString(it.GasUsed)

		txs = append(txs, types.Transaction{
			ChainID:      p.chainID,
			State:        state,
			Height:       height,
			Hash:         it.Hash,
			FromAddress:  it.From,
			ToAddress:    it.To,
			Balance:      valueRaw,
			Amount:       value,
			GasLimit:     gasLimit,
			GasUsed:      gasUsed,
			Type:         types.TxTypeInternal,
			CoinType:     types.CoinTypeInternal,
			Decimals:     types.NativeDefaultDecimals,
			CreatedTime:  unixTime,
			ModifiedTime: unixTime,
			TranType:     tranType,
		})
	}
	return txs
}
