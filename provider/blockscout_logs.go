package provider

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"strconv"
	"sync"
	"time"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// fetchBlockscoutLogs retrieves logs from Blockscout:
// GET /addresses/{address}/logs
func (t *BlockscoutProvider) fetchBlockscoutLogs(address string) (*model.BlockscoutLogResponse, error) {
	url := fmt.Sprintf("%s/addresses/%s/logs?limit=%d", t.config.URL, address, t.config.RequestPageSize)
	var result model.BlockscoutLogResponse
	if err := DoHttpRequestWithLogging("GET", "blockscout.logs", url, nil, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// fetchLogsByBlockFromRPC makes batched eth_getBlockReceipts requests for the
// given blocks and returns all logs indexed by txHash.
//
// Parameters
// -----------
// blocks : map[int64]struct{}
//
//	Set of block numbers to query. The keys are the block heights.
//
// Returns
// --------
// map[string][]model.BlockscoutLog
//
//	A map where the key is the transaction hash and the value is the slice
//	of logs that belong to that transaction.
//
// error
//
//	Non-nil if any shard fails. Partial results are discarded on error.
func (p *BlockscoutProvider) fetchLogsByBlockFromRPC(blocks map[int64]struct{}) (map[string][]model.BlockscoutLog, error) {
	if len(blocks) == 0 {
		return nil, nil
	}

	const (
		batchSize   = 50
		maxParallel = 4
	)
	requestTimeout := time.Duration(p.config.RPCRequestTimeout) * time.Second

	// Internal types for RPC structure
	type rpcReq struct {
		JSONRPC string        `json:"jsonrpc"`
		ID      int           `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}
	type receipt struct {
		TransactionHash string                `json:"transactionHash"`
		Logs            []model.BlockscoutLog `json:"logs"`
	}
	type rpcResp struct {
		ID     int       `json:"id"`
		Result []receipt `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	// Group blocks into batches (shards)
	var shardBlocks [][]int64
	cur := make([]int64, 0, batchSize)
	for blk := range blocks {
		cur = append(cur, blk)
		if len(cur) == batchSize {
			shardBlocks = append(shardBlocks, cur)
			cur = make([]int64, 0, batchSize)
		}
	}
	if len(cur) > 0 {
		shardBlocks = append(shardBlocks, cur)
	}

	// Prepare result container
	merged := make(map[string][]model.BlockscoutLog, 1024)
	var mu sync.Mutex

	// Launch parallel requests
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, maxParallel)

	for _, blocks := range shardBlocks {
		blkCopy := append([]int64(nil), blocks...) // capture range variable
		sem <- struct{}{}

		g.Go(func() error {
			defer func() { <-sem }()

			// Build the batch JSON-RPC request
			reqs := make([]rpcReq, 0, len(blkCopy))
			for i, b := range blkCopy {
				hexBlock := "0x" + strconv.FormatInt(b, 16)
				reqs = append(reqs, rpcReq{
					JSONRPC: "2.0",
					ID:      i + 1,
					Method:  "eth_getBlockReceipts",
					Params:  []interface{}{hexBlock},
				})
			}

			// Perform the HTTP POST using shared utility
			var rpcResponses []rpcResp
			if err := DoHttpRequestWithLogging(
				"POST",
				fmt.Sprintf("blockscout.rpcReceipts.shard.%d", len(blkCopy)),
				p.config.RPCURL,
				reqs,
				map[string]string{
					"Content-Type": "application/json",
				},
				&rpcResponses,
			); err != nil {
				return err
			}

			// Parse receipts and aggregate logs
			local := make(map[string][]model.BlockscoutLog, len(rpcResponses)*4)
			for _, r := range rpcResponses {
				if r.Error != nil {
					return fmt.Errorf("rpc error id=%d code=%d: %s", r.ID, r.Error.Code, r.Error.Message)
				}
				for _, rc := range r.Result {
					if len(rc.Logs) > 0 {
						local[rc.TransactionHash] = append(local[rc.TransactionHash], rc.Logs...)
					}
				}
			}

			// Safe merge to shared result map
			mu.Lock()
			for k, v := range local {
				merged[k] = append(merged[k], v...)
			}
			mu.Unlock()

			logger.Log.Debug().
				Int("blocks", len(blkCopy)).
				Int("tx_hashes", len(local)).
				Msg("Fetched logs shard successfully")

			return nil
		})
	}

	// Wait for all goroutines
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
