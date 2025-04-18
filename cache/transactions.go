// Package cache – transaction‑specific Redis routines.
package cache

import (
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"sync"
	"time"

	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// ParseTxAndSaveToCache groups a batch of transactions and writes them to
// Redis using pipelines / bulk commands for maximum throughput.
func (r *RedisCache) ParseTxAndSaveToCache(
	resp *model.TransactionResponse,
	address string,
) error {
	if resp == nil || len(resp.Result.Transactions) == 0 {
		logger.Log.Info().Msg("no transactions to cache")
		return nil
	}

	ttl := time.Duration(config.AppConfig.Cache.TTLSeconds) * time.Second
	logger.Log.Info().
		Int("txs", len(resp.Result.Transactions)).
		Dur("ttl", ttl).
		Msg("start caching")

	// -----------------------------------------------------------------------
	// 1. Grouping phase
	// -----------------------------------------------------------------------
	chainTxMap := make(map[int64][]model.Transaction)
	nativeTxMap := make(map[string][]model.Transaction)
	tokenTxMap := make(map[string][]model.Transaction)
	tokenSets := make(map[int64]map[string]struct{})

	for _, tx := range resp.Result.Transactions {
		chainTxMap[tx.ChainID] = append(chainTxMap[tx.ChainID], tx)

		chainName, err := config.ChainNameByID(tx.ChainID)
		if err != nil {
			logger.Log.Error().Err(err).Int64("chainID", tx.ChainID).Msg("chain name not found")
			continue
		}

		if tx.CoinType == model.CoinTypeNative {
			key := formatNativeKey(address, chainName)
			nativeTxMap[key] = append(nativeTxMap[key], tx)
		}

		if tx.CoinType == model.CoinTypeToken && tx.TokenAddress != "" {
			key := formatTokenKey(address, chainName, tx.TokenAddress)
			tokenTxMap[key] = append(tokenTxMap[key], tx)

			if tokenSets[tx.ChainID] == nil {
				tokenSets[tx.ChainID] = make(map[string]struct{})
			}
			tokenSets[tx.ChainID][tx.TokenAddress] = struct{}{}
		}
	}

	// -----------------------------------------------------------------------
	// 2. Writing phase (parallel)
	// -----------------------------------------------------------------------
	var wg sync.WaitGroup
	errCh := make(chan error, 8)

	// Helper to schedule JSON‑encoded pipelines.
	scheduleJSON := func(key string, txs []model.Transaction, label string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.SetJSONPipeline(key, txs, ttl); err != nil {
				logger.Log.Error().Err(err).Str("key", key).Msg("cache " + label + " failed")
				errCh <- err
				return
			}
			logger.Log.Debug().Str("key", key).Int("txs", len(txs)).Msg("cached " + label)
		}()
	}

	// Chain‑level sets (address‑chain).
	for chainID, txs := range chainTxMap {
		chainName, _ := config.ChainNameByID(chainID)
		scheduleJSON(formatChainKey(address, chainName), txs, "chainTx")
	}

	// Separate maps.
	for k, v := range nativeTxMap {
		scheduleJSON(k, v, "nativeTx")
	}
	for k, v := range tokenTxMap {
		scheduleJSON(k, v, "tokenTx")
	}

	// Token sets (SADD bulk).
	for chainID, tokenMap := range tokenSets {
		wg.Add(1)
		go func(chainID int64, tmap map[string]struct{}) {
			defer wg.Done()

			chainName, _ := config.ChainNameByID(chainID)
			setKey := formatTokenSetKey(address, chainName)

			// Map → slice.
			members := make([]string, 0, len(tmap))
			for token := range tmap {
				members = append(members, token)
			}
			if err := r.AddToSetBulk(setKey, members, ttl); err != nil {
				logger.Log.Error().Err(err).Str("setKey", setKey).Msg("cache token set failed")
				errCh <- err
				return
			}
			logger.Log.Debug().Str("setKey", setKey).Int("members", len(members)).Msg("cached token set")
		}(chainID, tokenMap)
	}

	// Wait & propagate first error, if any.
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	logger.Log.Info().Msg("all transactions cached")
	return nil
}

// QueryTxFromCache (unchanged except for minor style tweaks).
func (r *RedisCache) QueryTxFromCache(
	req *model.TransactionQueryParams,
) (*model.TransactionResponse, error) {
	var (
		out     = new(model.TransactionResponse)
		mu      sync.Mutex
		wg      sync.WaitGroup
		errChan = make(chan error, len(req.ChainNames))
	)

	if len(req.ChainNames) == 0 {
		logger.Log.Warn().Msg("no chain names given; skipping cache lookup")
		return out, nil
	}

	for _, chainName := range req.ChainNames {
		wg.Add(1)
		go func(chain string) {
			defer wg.Done()

			var key string
			if req.TokenAddress == "" {
				key = formatChainKey(req.Address, chain)
			} else {
				key = formatTokenKey(req.Address, chain, req.TokenAddress)
			}

			val, err := r.Get(key)
			if err != nil {
				errChan <- err
				return
			}

			var txs []model.Transaction
			if uErr := json.Unmarshal([]byte(val), &txs); uErr != nil {
				errChan <- uErr
				return
			}

			mu.Lock()
			out.Result.Transactions = append(out.Result.Transactions, txs...)
			mu.Unlock()
		}(chainName)
	}

	wg.Wait()
	close(errChan)
	// swallow individual cache misses; return first real error, if any
	for err := range errChan {
		if err != nil && err != redis.Nil {
			return out, err
		}
	}
	return out, nil
}
