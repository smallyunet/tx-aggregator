package cache

import (
	"encoding/json"
	"sync"
	"time"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// ParseTxAndSaveToCache processes transaction response and saves it to Redis cache in parallel
func (r *RedisCache) ParseTxAndSaveToCache(resp *model.TransactionResponse, address string) error {
	if resp == nil || len(resp.Result.Transactions) == 0 {
		logger.Log.Info().Msg("No transactions to process in response")
		return nil
	}

	logger.Log.Info().Int("transactionCount", len(resp.Result.Transactions)).Msg("Processing transactions for caching")

	chainTxMap := make(map[int64][]model.Transaction)
	nativeTxMap := make(map[string][]model.Transaction)
	tokenTxMap := make(map[string][]model.Transaction)
	tokenSetMap := make(map[int64]map[string]struct{})

	for _, tx := range resp.Result.Transactions {
		chainTxMap[tx.ChainID] = append(chainTxMap[tx.ChainID], tx)

		if tx.CoinType == model.CoinTypeNative {
			chainName, err := config.ChainNameByID(tx.ChainID)
			if err != nil {
				logger.Log.Error().Err(err).Int64("chainID", tx.ChainID).Msg("Failed to get chain name")
				continue
			}
			key := formatNativeKey(address, chainName)
			nativeTxMap[key] = append(nativeTxMap[key], tx)
		}

		if tx.CoinType == model.CoinTypeToken && tx.TokenAddress != "" {
			chainName, err := config.ChainNameByID(tx.ChainID)
			if err != nil {
				logger.Log.Error().Err(err).Int64("chainID", tx.ChainID).Msg("Failed to get chain name")
				continue
			}
			key := formatTokenKey(address, chainName, tx.TokenAddress)
			tokenTxMap[key] = append(tokenTxMap[key], tx)

			if _, exists := tokenSetMap[tx.ChainID]; !exists {
				tokenSetMap[tx.ChainID] = make(map[string]struct{})
			}
			tokenSetMap[tx.ChainID][tx.TokenAddress] = struct{}{}
		}
	}

	ttlSeconds := time.Duration(config.AppConfig.Cache.TTLSeconds) * time.Second
	logger.Log.Info().Int("cacheTTL", config.AppConfig.Cache.TTLSeconds).Msg("Setting cache TTL")

	var wg sync.WaitGroup
	errChan := make(chan error, len(chainTxMap)+len(tokenTxMap))

	for chainID, txs := range chainTxMap {
		wg.Add(1)
		go func(chainID int64, txs []model.Transaction) {
			defer wg.Done()
			chainName, err := config.ChainNameByID(chainID)
			if err != nil {
				logger.Log.Error().Err(err).Int64("chainID", chainID).Msg("Failed to get chain name")
				return
			}
			key := formatChainKey(address, chainName)
			logger.Log.Info().Int("txCount", len(txs)).Str("key", key).Msg("Caching chain transactions")
			if err := r.Set(key, txs, ttlSeconds); err != nil {
				logger.Log.Error().Err(err).Str("key", key).Msg("Failed to cache chain transactions")
				errChan <- err
			}
		}(chainID, txs)
	}

	for key, txs := range nativeTxMap {
		wg.Add(1)
		go func(key string, txs []model.Transaction) {
			defer wg.Done()
			logger.Log.Info().Int("txCount", len(txs)).Str("key", key).Msg("Caching native transactions")
			if err := r.Set(key, txs, ttlSeconds); err != nil {
				logger.Log.Error().Err(err).Str("key", key).Msg("Failed to cache native transactions")
				errChan <- err
			}
		}(key, txs)
	}

	for key, txs := range tokenTxMap {
		wg.Add(1)
		go func(key string, txs []model.Transaction) {
			defer wg.Done()
			logger.Log.Info().Int("txCount", len(txs)).Str("key", key).Msg("Caching token transactions")
			if err := r.Set(key, txs, ttlSeconds); err != nil {
				logger.Log.Error().Err(err).Str("key", key).Msg("Failed to cache token transactions")
				errChan <- err
			}
		}(key, txs)
	}

	for chainID, tokens := range tokenSetMap {
		wg.Add(1)
		go func(chainID int64, tokens map[string]struct{}) {
			defer wg.Done()
			chainName, err := config.ChainNameByID(chainID)
			if err != nil {
				logger.Log.Error().Err(err).Int64("chainID", chainID).Msg("Failed to get chain name")
				return
			}
			setKey := formatTokenSetKey(address, chainName)
			logger.Log.Info().Int("tokenCount", len(tokens)).Str("setKey", setKey).Msg("Caching token set")
			for token := range tokens {
				if err := r.AddToSet(setKey, token, ttlSeconds); err != nil {
					logger.Log.Error().Err(err).Str("token", token).Str("setKey", setKey).Msg("Failed to cache token set")
					errChan <- err
					return
				}
			}
		}(chainID, tokens)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	logger.Log.Info().Msg("Successfully cached all transactions")
	return nil
}

// QueryTxFromCache retrieves transactions from cache in parallel based on query parameters.
func (r *RedisCache) QueryTxFromCache(req *model.TransactionQueryParams) (*model.TransactionResponse, error) {
	resp := new(model.TransactionResponse)

	logger.Log.Debug().
		Strs("chainIDs", req.ChainNames).
		Str("tokenAddress", req.TokenAddress).
		Str("address", req.Address).
		Msg("Querying cache")

	if len(req.ChainNames) == 0 {
		logger.Log.Warn().Msg("No chainIDs provided, skipping cache lookup")
		return resp, nil
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(req.ChainNames))

	for _, chainName := range req.ChainNames {
		wg.Add(1)
		go func(chainName string) {
			defer wg.Done()

			var key string
			if req.TokenAddress == "" {
				key = formatChainKey(req.Address, chainName)
			} else {
				key = formatTokenKey(req.Address, chainName, req.TokenAddress)
			}

			val, err := r.Get(key)
			if err != nil {
				logger.Log.Debug().
					Str("address", req.Address).
					Str("chainName", chainName).
					Str("key", key).
					Err(err).
					Msg("Cache not found or failed to get")
				errChan <- err
				return
			}

			var txs []model.Transaction
			if err := json.Unmarshal([]byte(val), &txs); err != nil {
				logger.Log.Warn().
					Str("address", req.Address).
					Str("chainName", chainName).
					Str("key", key).
					Err(err).
					Msg("Failed to unmarshal transactions from cache")
				errChan <- err
				return
			}

			logger.Log.Debug().
				Str("address", req.Address).
				Str("chainName", chainName).
				Str("key", key).
				Int("txCount", len(txs)).
				Msg("Retrieved transactions from cache")

			mu.Lock()
			resp.Result.Transactions = append(resp.Result.Transactions, txs...)
			mu.Unlock()
		}(chainName)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			logger.Log.Warn().Err(err).Msg("Some queries failed in parallel")
		}
	}

	logger.Log.Info().
		Int("totalTxCount", len(resp.Result.Transactions)).
		Msg("Finished querying cache")

	return resp, nil
}
