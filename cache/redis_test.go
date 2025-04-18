package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// helper: create RedisCache using a fresh miniredis instance.
// The server is automatically closed when the test finishes.
func newTestRedisCache(t *testing.T) *RedisCache {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	// Ensure the in‑memory server is stopped after the test.
	t.Cleanup(s.Close)

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	return &RedisCache{
		client: client,
		ctx:    context.Background(),
		mode:   "single",
	}
}

func TestSetJSONPipeline(t *testing.T) {
	cache := newTestRedisCache(t)

	type dummy struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	key := "user:1"
	value := dummy{Name: "Alice", Age: 30}
	ttl := 10 * time.Second

	err := cache.SetJSONPipeline(key, value, ttl)
	assert.NoError(t, err)

	got, err := cache.Get(key)
	assert.NoError(t, err)
	assert.Contains(t, got, "Alice")
}

func TestAddToSetBulk(t *testing.T) {
	cache := newTestRedisCache(t)

	key := "myset"
	members := []string{"a", "b", "c"}
	ttl := 5 * time.Second

	err := cache.AddToSetBulk(key, members, ttl)
	assert.NoError(t, err)

	// Check each member exists
	for _, m := range members {
		isMember, err := cache.client.SIsMember(cache.ctx, key, m).Result()
		assert.NoError(t, err)
		assert.True(t, isMember)
	}
}

func TestGet_NotFound(t *testing.T) {
	cache := newTestRedisCache(t)

	_, err := cache.Get("nonexistent")
	assert.Error(t, err)
}
