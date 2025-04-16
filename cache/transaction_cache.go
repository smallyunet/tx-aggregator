package cache

import (
	"encoding/json"
	"log"
	"time"
	"tx-aggregator/config"
	"tx-aggregator/model"
	"tx-aggregator/types"
)

// ParseTxAndSaveToCache processes transaction response and saves it to Redis cache
// It groups transactions by chainId and tokenAddress, then stores them in appropriate cache keys
// Returns error if any cache operation fails
func (r *RedisCache) ParseTxAndSaveToCache(resp *model.TransactionResponse) error {
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
			key := formatTokenKey(tx.ChainID, tx.TokenAddress)
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

	// Save chain-level transactions
	for chainID, txs := range chainTxMap {
		key := formatChainKey(chainID)
		log.Printf("Caching %d transactions for chain %d", len(txs), chainID)
		if err := r.Set(key, txs, ttlSeconds); err != nil {
			log.Printf("Failed to cache transactions for chain %d: %v", chainID, err)
			return err
		}
	}

	// Save token-specific transactions
	for key, txs := range tokenTxMap {
		log.Printf("Caching %d token transactions for key %s", len(txs), key)
		if err := r.Set(key, txs, ttlSeconds); err != nil {
			log.Printf("Failed to cache token transactions for key %s: %v", key, err)
			return err
		}
	}

	// Save token sets for each chainId
	for chainID, tokens := range tokenSetMap {
		setKey := formatTokenSetKey(chainID)
		log.Printf("Caching %d tokens for chain %d", len(tokens), chainID)
		for token := range tokens {
			if err := r.AddToSet(setKey, token, ttlSeconds); err != nil {
				log.Printf("Failed to cache token set for chain %d: %v", chainID, err)
				return err
			}
		}
	}

	log.Println("Successfully cached all transactions")
	return nil
}

// QueryTxFromCache retrieves transactions from cache based on query parameters
// It supports querying by chainId and chainId-tokenAddress combinations
// Returns TransactionResponse containing matched transactions
func (r *RedisCache) QueryTxFromCache(req *types.TransactionQueryParams) (*model.TransactionResponse, error) {
	resp := new(model.TransactionResponse)
	log.Printf("Querying cache for chainIDs: %v, tokenAddress: %s", req.ChainIDs, req.TokenAddress)

	if req.TokenAddress == "" {
		// Query by chainId only
		for _, chainID := range req.ChainIDs {
			key := formatChainKey(chainID)
			val, err := r.Get(key)
			if err != nil {
				log.Printf("No cache found for chain %d: %v", chainID, err)
				continue
			}

			var txs []model.Transaction
			if err := json.Unmarshal([]byte(val), &txs); err != nil {
				log.Printf("Failed to unmarshal transactions for chain %d: %v", chainID, err)
				continue
			}

			log.Printf("Retrieved %d transactions for chain %d", len(txs), chainID)
			resp.Result.Transactions = append(resp.Result.Transactions, txs...)
		}
	}

	if req.TokenAddress != "" {
		// Query by chainId-tokenAddress combination
		for _, chainID := range req.ChainIDs {
			key := formatTokenKey(chainID, req.TokenAddress)
			val, err := r.Get(key)
			if err != nil {
				log.Printf("No cache found for chain %d token %s: %v", chainID, req.TokenAddress, err)
				continue
			}

			var txs []model.Transaction
			if err := json.Unmarshal([]byte(val), &txs); err != nil {
				log.Printf("Failed to unmarshal token transactions for chain %d: %v", chainID, err)
				continue
			}

			log.Printf("Retrieved %d token transactions for chain %d token %s", len(txs), chainID, req.TokenAddress)
			resp.Result.Transactions = append(resp.Result.Transactions, txs...)
		}
	}

	log.Printf("Total transactions retrieved from cache: %d", len(resp.Result.Transactions))
	return resp, nil
}
