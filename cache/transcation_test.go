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
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	rc := newRedisCacheWithServer(t, s)

	// Prepare input data
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

	// Inject config for test
	cfg := config.Current()
	cfg.Redis.TTLSeconds = 100
	cfg.ChainNames = map[string]int64{"ETH": 1}
	config.SetCurrentConfig(cfg)

	// Call function under test
	err = rc.ParseTxAndSaveToCache(resp, "0xUser")
	assert.NoError(t, err)

	// Verify keys written to Redis
	keys := s.Keys()
	t.Logf("Keys in miniredis: %v", keys)
	assert.NotEmpty(t, keys)
}

func TestQueryTxFromCache_Integration(t *testing.T) {
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	rc := newRedisCacheWithServer(t, s)

	// Inject mock config for chain ID ↔ name resolution
	cfg := config.Current()
	cfg.ChainNames = map[string]int64{"ETH": 1}
	config.SetCurrentConfig(cfg)

	// Pre-write fake data
	key := formatChainKey("0xUser", "ETH")
	txs := []types.Transaction{
		{
			Hash:    "0xabc",
			ChainID: 1,
		},
	}
	bz, _ := json.Marshal(txs)
	s.Set(key, string(bz))
	s.SetTTL(key, time.Hour)

	// Query the cache
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
