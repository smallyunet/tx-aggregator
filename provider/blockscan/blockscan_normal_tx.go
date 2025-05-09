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

func (p *BlockscanProvider) fetchNormalTx(addr string) (*types.BlockscanNormalTxResp, error) {
	q := url.Values{
		"module":     {"account"},
		"action":     {"txlist"},
		"address":    {addr},
		"startblock": {strconv.FormatInt(p.cfg.Startblock, 10)},
		"endblock":   {strconv.FormatInt(p.cfg.Endblock, 10)},
		"page":       {strconv.FormatInt(p.cfg.Page, 10)},
		"offset":     {fmt.Sprint(p.cfg.RequestPageSize)},
		"sort":       {p.cfg.Sort},
		"apikey":     {p.cfg.APIKey},
	}
	var out types.BlockscanNormalTxResp
	u := fmt.Sprintf("%s?%s", p.cfg.URL, q.Encode())
	if err := utils.DoHttpRequestWithLogging("GET", "blockscan.normalTx", u, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *BlockscanProvider) transformNormalTx(resp *types.BlockscanNormalTxResp, address string) []types.Transaction {
	if resp == nil || resp.Status != "1" || len(resp.Result) == 0 {
		return nil
	}

	var txs []types.Transaction
	for _, it := range resp.Result {
		height := utils.ParseStringToInt64OrDefault(it.BlockNumber, 0)
		unixTime := utils.ParseStringToInt64OrDefault(it.TimeStamp, 0)
		txIndex := utils.ParseStringToInt64OrDefault(it.TransactionIndex, 0)

		state := types.TxStateFail
		if it.IsError == "0" && it.TxReceiptStatus == "1" {
			state = types.TxStateSuccess
		}

		tranType := types.TransTypeOut
		if strings.EqualFold(it.To, address) {
			tranType = types.TransTypeIn
		}

		amountRaw, _ := utils.NormalizeNumericString(it.Value)
		amount := utils.DivideByDecimals(amountRaw, types.NativeDefaultDecimals)
		gasLimit, _ := utils.NormalizeNumericString(it.Gas)
		gasUsed, _ := utils.NormalizeNumericString(it.GasUsed)
		gasPrice, _ := utils.NormalizeNumericString(it.GasPrice)
		nonce, _ := utils.NormalizeNumericString(it.Nonce)

		nativeSymbol, err := utils.NativeTokenByChainID(p.chainID)
		if err != nil {
			logger.Log.Error().
				Err(err).
				Int64("chain_id", p.chainID).
				Msg("Failed to get native token name")
		}

		txs = append(txs, types.Transaction{
			ChainID:          p.chainID,
			State:            state,
			Height:           height,
			Hash:             it.Hash,
			BlockHash:        it.BlockHash,
			TxIndex:          txIndex,
			FromAddress:      it.From,
			ToAddress:        it.To,
			TokenAddress:     "",
			Balance:          amountRaw,
			Amount:           amount,
			GasLimit:         gasLimit,
			GasUsed:          gasUsed,
			GasPrice:         gasPrice,
			Nonce:            nonce,
			Type:             types.TxTypeUnknown, // native transfer
			CoinType:         types.CoinTypeNative,
			TokenDisplayName: nativeSymbol,
			Decimals:         types.NativeDefaultDecimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
		})
	}
	return txs
}
