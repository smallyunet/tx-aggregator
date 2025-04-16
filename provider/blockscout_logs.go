package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// fetchBlockscoutLogs retrieves logs from Blockscout:
// GET /addresses/{address}/logs
func (t *BlockscoutProvider) fetchBlockscoutLogs(address string) (*model.BlockscoutLogResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/logs", t.baseURL, address)
	logger.Log.Debug().Str("url", url).Msg("Fetching logs from Blockscout")

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs response: %w", err)
	}

	var result model.BlockscoutLogResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %w", err)
	}

	return &result, nil
}

// fetchLogsByBlockFromRPC makes a batch request to the RPC node: eth_getBlockReceipts for each block number
// and returns a map of txHash => []BlockscoutLog
func (t *BlockscoutProvider) fetchLogsByBlockFromRPC(blocks map[int64]bool) (map[string][]model.BlockscoutLog, error) {
	if len(blocks) == 0 {
		return nil, nil
	}

	// 1. Construct batch requests
	var rpcRequests []map[string]interface{}
	idCount := 1
	for block := range blocks {
		hexBlock := "0x" + strconv.FormatInt(block, 16)
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      idCount,
			"method":  "eth_getBlockReceipts",
			"params":  []interface{}{hexBlock},
		}
		rpcRequests = append(rpcRequests, req)
		idCount++
	}

	reqBody, err := json.Marshal(rpcRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch RPC requests: %w", err)
	}

	// 2. Send HTTP POST request
	resp, err := http.Post(t.rpcURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block receipts from RPC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received non-success status code from RPC: %d", resp.StatusCode)
	}

	// 3. Parse batch response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RPC response: %w", err)
	}

	var rpcRespList []model.RpcReceiptResponse
	if err := json.Unmarshal(body, &rpcRespList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal RPC receipts: %w", err)
	}

	// 4. Organize into txHash => []BlockscoutLog
	resultMap := make(map[string][]model.BlockscoutLog)
	for _, blockResp := range rpcRespList {
		for _, receipt := range blockResp.Result {
			// Iterate through the logs of this receipt
			for _, l := range receipt.Logs {
				// Convert to BlockscoutLog format (only fields that will be used later)
				tmp := model.BlockscoutLog{
					Address: model.BlockscoutAddressDetails{
						Hash: l.Address,
					},
					BlockHash:       l.BlockHash,
					Data:            l.Data,
					Topics:          l.Topics,
					TransactionHash: l.TransactionHash,
				}

				resultMap[l.TransactionHash] = append(resultMap[l.TransactionHash], tmp)
			}
		}
	}

	return resultMap, nil
}

// indexBlockscoutLogsByTxHash stores each log in a map keyed by transaction hash.
func (t *BlockscoutProvider) indexBlockscoutLogsByTxHash(resp *model.BlockscoutLogResponse) map[string][]model.BlockscoutLog {
	logsMap := make(map[string][]model.BlockscoutLog)
	if resp == nil || len(resp.Items) == 0 {
		return logsMap
	}

	for _, lg := range resp.Items {
		txHash := lg.TransactionHash
		logsMap[txHash] = append(logsMap[txHash], lg)
	}
	return logsMap
}
