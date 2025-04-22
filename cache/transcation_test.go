package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"tx-aggregator/config"
	"tx-aggregator/types"
)

// newRedisCacheWithServer builds a RedisCache that communicates with the provided miniredis server instance.
func newRedisCacheWithServer(t *testing.T, s *miniredis.Miniredis) *RedisCache {
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	return &RedisCache{
		client: client,
		ctx:    context.Background(),
		mode:   "single",
	}
}

func TestParseTxAndSaveToCache_Integration(t *testing.T) {
	// ① share the same miniredis instance
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	rc := newRedisCacheWithServer(t, s)

	// ② prepare input data
	resp := &types.TransactionResponse{}
	resp.Result.Transactions = []types.Transaction{
		{
			ChainID:  1,
			CoinType: types.CoinTypeNative,
		},
		{
			ChainID:      1,
			CoinType:     types.CoinTypeToken,
			TokenAddress: "0xToken",
		},
	}

	// ③ mock configuration
	config.AppConfig.Cache.TTLSeconds = 100
	config.AppConfig.ChainNames = map[string]int64{"ETH": 1}

	err = rc.ParseTxAndSaveToCache(resp, "0xUser")
	assert.NoError(t, err)

	// ④ assert that data was written successfully
	keys := s.Keys() // read directly from the same miniredis instance
	t.Logf("Keys in miniredis: %v", keys)
	assert.NotEmpty(t, keys)
}

func TestQueryTxFromCache_Integration(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	rc := newRedisCacheWithServer(t, s)

	// ① pre-write fake data
	key := formatChainKey("0xUser", "ETH") // function should have been refactored to use chain name
	txs := []types.Transaction{
		{
			Hash:    "0xabc",
			ChainID: 1,
		},
	}
	bz, _ := json.Marshal(txs)
	s.Set(key, string(bz))   // write directly to miniredis
	s.SetTTL(key, time.Hour) // ensure the key does not expire

	// ② query the cache
	params := &types.TransactionQueryParams{
		Address:    "0xUser",
		ChainNames: []string{"ETH"},
	}

	resp, err := rc.QueryTxFromCache(params)
	assert.NoError(t, err)
	assert.Len(t, resp.Result.Transactions, 1)
	assert.Equal(t, "0xabc", resp.Result.Transactions[0].Hash)
}

func TestQueryTxFromCache_EmptyChains(t *testing.T) {
	// use an empty (unconnected Redis) RedisCache to verify logic for empty chain names
	rc := &RedisCache{
		client: nil,
		ctx:    context.Background(),
	}

	params := &types.TransactionQueryParams{
		Address:    "0xUser",
		ChainNames: []string{},
	}

	resp, err := rc.QueryTxFromCache(params)
	assert.NoError(t, err)
	assert.Empty(t, resp.Result.Transactions)
}
