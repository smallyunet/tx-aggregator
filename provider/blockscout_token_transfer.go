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

// fetchBlockscoutTokenTransfers retrieves token transfers from Blockscout:
// GET /addresses/{address}/token-transfers
func (t *BlockscoutProvider) fetchBlockscoutTokenTransfers(address string) (*model.BlockscoutTokenTransferResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/token-transfers", t.baseURL, address)
	logger.Log.Debug().Str("url", url).Msg("Fetching token transfers from Blockscout")

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token transfers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token transfers response: %w", err)
	}

	var result model.BlockscoutTokenTransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token transfers: %w", err)
	}

	return &result, nil
}

// transformBlockscoutTokenTransfers converts Blockscout token transfers into []model.Transaction
func (t *BlockscoutProvider) transformBlockscoutTokenTransfers(resp *model.BlockscoutTokenTransferResponse, address string) []model.Transaction {
	if resp == nil || len(resp.Items) == 0 {
		logger.Log.Warn().Msg("No token transfers to transform from Blockscout")
		return nil
	}

	var transactions []model.Transaction
	for _, tt := range resp.Items {
		// Transaction direction: Outgoing by default
		tranType := model.TransTypeOut
		if strings.EqualFold(tt.To.Hash, address) {
			tranType = model.TransTypeIn
		}

		unixTime := parseBlockscoutTimestampToUnix(tt.Timestamp)
		// Always consider token transfers state = 1 if there's no explicit fail indicator
		state := 1

		// Parse decimals from token info
		decimals := parseStringToInt64OrDefault(tt.Token.Decimals, 18) // default 18 if parse fails

		transactions = append(transactions, model.Transaction{
			ChainID:          t.chainID,
			State:            state,
			Height:           tt.BlockNumber,
			Hash:             tt.TransactionHash,
			BlockHash:        tt.BlockHash,
			FromAddress:      tt.From.Hash,
			ToAddress:        tt.To.Hash,
			TokenAddress:     tt.Token.Address,
			Amount:           tt.Total.Value,       // the raw string in subunits
			GasUsed:          "",                   // not provided in token transfers
			GasLimit:         "",                   // not provided
			GasPrice:         "",                   // not provided
			Nonce:            "",                   // not provided
			Type:             model.TxTypeTransfer, // 0 = normal, 1 = approve, etc.
			CoinType:         model.CoinTypeToken,  // 2 = token
			TokenDisplayName: tt.Token.Name,
			Decimals:         decimals,
			CreatedTime:      unixTime,
			ModifiedTime:     unixTime,
			TranType:         tranType,
			ApproveShow:      "",
			IconURL:          tt.Token.IconURL,
		})
	}

	logger.Log.Debug().
		Int("transformed_count", len(transactions)).
		Msg("Transformed token transfers from Blockscout")
	return transactions
}
