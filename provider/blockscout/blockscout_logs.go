package blockscout

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"strconv"
	"sync"
	"time"
	"tx-aggregator/logger"
	"tx-aggregator/model"
	"tx-aggregator/provider"
)

// fetchBlockscoutLogs retrieves logs from Blockscout:
// GET /addresses/{address}/logs
func (t *BlockscoutProvider) fetchBlockscoutLogs(address string) (*model.BlockscoutLogResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/logs?limit=%d", t.config.URL, address, t.config.RequestPageSize)
	var result model.BlockscoutLogResponse
	if err := provider.DoHttpRequestWithLogging("GET", "blockscout.logs", url, nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// fetchLogsByBlockFromRPC issues batched eth_getBlockReceipts requests, converts
// the raw RPC receipts into Blockscout‑style logs, and returns them grouped by
// transaction hash.
//
// ───────────────────────────────────────────────────────────────────────────────
// blocks        Set of block numbers to query (map key = height, value ignored)
// return.value  map[txHash][]model.BlockscoutLog
// return.error  Non‑nil if any shard fails (partial results are discarded)
// ───────────────────────────────────────────────────────────────────────────────
func (p *BlockscoutProvider) fetchLogsByBlockFromRPC(
	blocks map[int64]struct{},
) (map[string][]model.BlockscoutLog, error) {

	if len(blocks) == 0 {
		return nil, nil
	}

	// Tune these to your infra.
	const (
		batchSize   = 50 // how many blocks per JSON‑RPC batch request
		maxParallel = 4  // how many batches in flight at once
	)
	reqTimeout := time.Duration(p.config.RPCRequestTimeout) * time.Second

	// ───────────────────────────── Split into shards ──────────────────────────
	var shards [][]int64
	buf := make([]int64, 0, batchSize)
	for blk := range blocks {
		buf = append(buf, blk)
		if len(buf) == batchSize {
			shards = append(shards, buf)
			buf = make([]int64, 0, batchSize)
		}
	}
	if len(buf) > 0 {
		shards = append(shards, buf)
	}

	merged := make(map[string][]model.BlockscoutLog, 1024) // final result
	var mu sync.Mutex                                      // guards merged

	// Cancellation context for all HTTP calls
	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, maxParallel) // simple semaphore

	for _, shard := range shards {
		shard := append([]int64(nil), shard...) // capture range var
		sem <- struct{}{}                       // acquire slot

		g.Go(func() error {
			defer func() { <-sem }() // release slot

			// ─────────────── Build JSON‑RPC batch request payload ──────────────
			type rpcReq struct {
				JSONRPC string        `json:"jsonrpc"`
				ID      int           `json:"id"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
			}
			reqs := make([]rpcReq, 0, len(shard))
			for i, blk := range shard {
				reqs = append(reqs, rpcReq{
					JSONRPC: "2.0",
					ID:      i + 1,
					Method:  "eth_getBlockReceipts",
					Params:  []interface{}{"0x" + strconv.FormatInt(blk, 16)},
				})
			}

			// ─────────────── Send HTTP POST & parse into model structs ─────────
			var rpcResponses []model.RpcReceiptResponse
			if err := provider.DoHttpRequestWithLogging(
				"POST",
				fmt.Sprintf("blockscout.rpcReceipts.shard.%d", len(shard)),
				p.config.RPCURL,
				reqs,
				map[string]string{"Content-Type": "application/json"},
				&rpcResponses,
			); err != nil {
				return err
			}

			// ─────────── Convert RpcReceiptLog → BlockscoutLog ────────────────
			local := make(map[string][]model.BlockscoutLog, len(rpcResponses)*4)

			for _, resp := range rpcResponses {
				for _, receipt := range resp.Result {
					for _, l := range receipt.Logs {

						// Convert hex strings → int64 where needed
						var (
							blockNum int64
							idx      int64
						)
						if len(l.BlockNumber) > 2 { // "0x..."
							if v, err := strconv.ParseInt(l.BlockNumber[2:], 16, 64); err == nil {
								blockNum = v
							}
						}
						if len(l.LogIndex) > 2 {
							if v, err := strconv.ParseInt(l.LogIndex[2:], 16, 64); err == nil {
								idx = v
							}
						}

						log := model.BlockscoutLog{
							Address: model.BlockscoutAddressDetails{
								Hash: l.Address,
							},
							BlockHash:       l.BlockHash,
							BlockNumber:     blockNum,
							Data:            l.Data,
							Topics:          l.Topics,
							TransactionHash: l.TransactionHash,
							Index:           idx,
							// SmartContract / Decoded will remain zero‑value
						}
						local[l.TransactionHash] = append(local[l.TransactionHash], log)
					}
				}
			}

			// ─────────────── Thread‑safe merge into final map ──────────────────
			mu.Lock()
			for txHash, logs := range local {
				merged[txHash] = append(merged[txHash], logs...)
			}
			mu.Unlock()

			logger.Log.Debug().
				Int("blocks", len(shard)).
				Int("tx_hashes", len(local)).
				Msg("Fetched logs shard successfully")

			return nil
		})
	}

	// Wait for every goroutine. If any returns error, whole call fails.
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return merged, nil
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
