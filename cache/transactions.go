package cache

import (
	"encoding/json"
	"log"
	"sync"
	"time"
	"tx-aggregator/config"
	"tx-aggregator/logger"
	"tx-aggregator/model"
)

// ParseTxAndSaveToCache processes transaction response and saves it to Redis cache in parallel
// It groups transactions by chainId and tokenAddress, then stores them in appropriate cache keys
// Returns error if any cache operation fails
func (r *RedisCache) ParseTxAndSaveToCache(resp *model.TransactionResponse, address string) error {
	if resp == nil || len(resp.Result.Transactions) == 0 {
		log.Println("No transactions to process in response")
		return nil
	}

	log.Printf("Processing %d transactions for caching", len(resp.Result.Transactions))

	// Initialize maps for grouping transactions
	chainTxMap := make(map[int64][]model.Transaction)  // chainId -> transactions
	tokenTxMap := make(map[string][]model.Transaction) // chainId-tokenAddress -> transactions
	tokenSetMap := make(map[int64]map[string]struct{}) // chainId -> set of token addresses

	// Group transactions by chainId and tokenAddress
	for _, tx := range resp.Result.Transactions {
		// Append to chainId group
		chainTxMap[tx.ChainID] = append(chainTxMap[tx.ChainID], tx)

		// If token transaction, append to chainId-tokenAddress group
		if tx.CoinType == 2 && tx.TokenAddress != "" {
			key := formatTokenKey(address, tx.ChainID, tx.TokenAddress)
			tokenTxMap[key] = append(tokenTxMap[key], tx)

			// Track token set for the chain
			if _, exists := tokenSetMap[tx.ChainID]; !exists {
				tokenSetMap[tx.ChainID] = make(map[string]struct{})
			}
			tokenSetMap[tx.ChainID][tx.TokenAddress] = struct{}{}
		}
	}

	ttlSeconds := time.Duration(config.AppConfig.Cache.TTLSeconds) * time.Second
	log.Printf("Setting cache TTL to %d seconds", config.AppConfig.Cache.TTLSeconds)

	// Use WaitGroup for parallel caching
	var wg sync.WaitGroup
	// Use a buffered channel to capture errors from goroutines
	errChan := make(chan error, len(chainTxMap)+len(tokenTxMap))

	// Save chain-level transactions in parallel
	for chainID, txs := range chainTxMap {
		wg.Add(1)
		go func(chainID int64, txs []model.Transaction) {
			defer wg.Done()
			key := formatChainKey(address, chainID)
			log.Printf("Caching %d transactions for chain %d", len(txs), chainID)
			if err := r.Set(key, txs, ttlSeconds); err != nil {
				log.Printf("Failed to cache transactions for chain %d: %v", chainID, err)
				errChan <- err
			}
		}(chainID, txs)
	}

	// Save token-specific transactions in parallel
	for key, txs := range tokenTxMap {
		wg.Add(1)
		go func(key string, txs []model.Transaction) {
			defer wg.Done()
			log.Printf("Caching %d token transactions for key %s", len(txs), key)
			if err := r.Set(key, txs, ttlSeconds); err != nil {
				log.Printf("Failed to cache token transactions for key %s: %v", key, err)
				errChan <- err
			}
		}(key, txs)
	}

	// Save token sets in parallel
	// To simplify, writing tokenSetMap is done serially by chainID
	// If you want to further parallelize writing tokens within each chainID, you can add nested goroutines
	for chainID, tokens := range tokenSetMap {
		wg.Add(1)
		go func(chainID int64, tokens map[string]struct{}) {
			defer wg.Done()
			setKey := formatTokenSetKey(address, chainID)
			log.Printf("Caching %d tokens for chain %d", len(tokens), chainID)
			for token := range tokens {
				if err := r.AddToSet(setKey, token, ttlSeconds); err != nil {
					log.Printf("Failed to cache token set for chain %d: %v", chainID, err)
					errChan <- err
					return
				}
			}
		}(chainID, tokens)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	// Close channel to avoid goroutine leaks
	close(errChan)

	// Check if any error occurred
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	log.Println("Successfully cached all transactions")
	return nil
}

// QueryTxFromCache retrieves transactions from cache in parallel based on query parameters.
// It supports querying by chainId or chainId-tokenAddress combinations.
// If one cache lookup fails, it continues to check the others.
func (r *RedisCache) QueryTxFromCache(req *model.TransactionQueryParams) (*model.TransactionResponse, error) {
	resp := new(model.TransactionResponse)

	logger.Log.Debug().
		Ints64("chainIDs", req.ChainIDs).
		Str("tokenAddress", req.TokenAddress).
		Str("address", req.Address).
		Msg("Querying cache")

	if len(req.ChainIDs) == 0 {
		logger.Log.Warn().Msg("No chainIDs provided, skipping cache lookup")
		return resp, nil
	}

	// We'll collect transactions safely from goroutines
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(req.ChainIDs))

	for _, chainID := range req.ChainIDs {
		wg.Add(1)
		go func(chainID int64) {
			defer wg.Done()

			var key string
			if req.TokenAddress == "" {
				// Query by chainId only
				key = formatChainKey(req.Address, chainID)
			} else {
				// Query by chainId-tokenAddress combination
				key = formatTokenKey(req.Address, chainID, req.TokenAddress)
			}

			val, err := r.Get(key)
			if err != nil {
				logger.Log.Debug().
					Str("address", req.Address).
					Int64("chainID", chainID).
					Str("key", key).
					Err(err).
					Msg("Cache not found or failed to get")
				// Not returning immediately; just log & collect error
				errChan <- err
				return
			}

			var txs []model.Transaction
			if err := json.Unmarshal([]byte(val), &txs); err != nil {
				logger.Log.Warn().
					Str("address", req.Address).
					Int64("chainID", chainID).
					Str("key", key).
					Err(err).
					Msg("Failed to unmarshal transactions from cache")
				errChan <- err
				return
			}

			logger.Log.Debug().
				Str("address", req.Address).
				Int64("chainID", chainID).
				Str("key", key).
				Int("txCount", len(txs)).
				Msg("Retrieved transactions from cache")

			// Lock and append to the response
			mu.Lock()
			resp.Result.Transactions = append(resp.Result.Transactions, txs...)
			mu.Unlock()
		}(chainID)
	}

	// Wait for all queries to finish
	wg.Wait()
	// Close channel to avoid goroutine leaks
	close(errChan)

	// Check if any error occurred
	for err := range errChan {
		if err != nil {
			// You can choose to return only the first error, or accumulate all errors
			// If you want to ignore all concurrent errors, just log and do not return
			// In this example, we return the first error
			logger.Log.Warn().
				Err(err).
				Msg("Some queries failed in parallel")
		}
	}

	logger.Log.Info().
		Int("totalTxCount", len(resp.Result.Transactions)).
		Msg("Finished querying cache")

	return resp, nil
}
