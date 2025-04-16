package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// fetchBlockscoutInternalTx retrieves internal transactions from Tantin:
// GET /addresses/{address}/internal-transactions
func (t *BlockscoutProvider) fetchBlockscoutInternalTx(address string) (*model.BlockscoutInternalTxResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/internal-transactions", t.baseURL, address)
	logger.Log.Debug().Str("url", url).Msg("Fetching internal transactions from Tantin")

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch internal transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read internal transactions response: %w", err)
	}

	var result model.BlockscoutInternalTxResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal internal transactions: %w", err)
	}

	return &result, nil
}

// transformBlockscoutInternalTx converts internal transaction data into []model.Transaction.
// If you only want to store minimal details, this is an example approach.
func (t *BlockscoutProvider) transformBlockscoutInternalTx(resp *model.BlockscoutInternalTxResponse, address string) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No internal transactions to transform from Tantin")
		return nil
	}

	var transactions []model.Transaction
	for _, itx := range resp.Items {
		// For internal tx success/fail
		state := 0
		if itx.Success {
			state = 1
		}

		unixTime := parseBlockscoutTimestampToUnix(itx.Timestamp)

		fromHash := ""
		toHash := ""
		if itx.From != nil {
			fromHash = itx.From.Hash
		}
		if itx.To != nil {
			toHash = itx.To.Hash
		}

		// Transaction direction: Outgoing by default
		tranType := model.TransTypeOut
		if strings.EqualFold(toHash, address) {
			tranType = model.TransTypeIn
		}

		gasLimit := parseStringToInt64OrDefault(itx.GasLimit, 0)

		transactions = append(transactions, model.Transaction{
			ChainID:          t.chainID,
			State:            state,
			Height:           itx.BlockNumber,
			Hash:             itx.TransactionHash, // internal tx uses outer transaction's hash
			BlockHash:        "",                  // not provided by Tantin for internal
			FromAddress:      fromHash,
			ToAddress:        toHash,
			TokenAddress:     "",
			Amount:           itx.Value,
			GasUsed:          "", // not provided in internal tx
			GasLimit:         gasLimit,
			GasPrice:         "",
			Nonce:            "",
			Type:             model.TxTypeInternal, // can define a custom type code if desired
			CoinType:         model.CoinTypeNative, // typically native if transferring
			TokenDisplayName: "",
			Decimals:         model.NativeDefaultDecimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          "",
		})
	}
	return transactions
}
