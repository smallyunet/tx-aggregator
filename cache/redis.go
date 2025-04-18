// Package cache wraps go‑redis and provides a few pipeline‑based helpers
// to speed up high‑volume writes for the tx‑aggregator service.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"tx-aggregator/logger"
)

// RedisCache is a thin wrapper around a go‑redis client.  It works for both
// single‑instance and cluster deployments.
type RedisCache struct {
	client redis.Cmdable   // *redis.Client or *redis.ClusterClient
	ctx    context.Context // shared context for all calls
	mode   string          // "single" or "cluster" (for debugging only)
}

// NewRedisCache detects whether the target is a single node or a cluster
// from the number of addresses provided and initialises the appropriate
// client.  Pool settings are tuned for high concurrency.
func NewRedisCache(addrs []string, password string) *RedisCache {
	const (
		poolSize    = 40 // adjust to your workload (≈ 10 × CPU cores)
		minIdleConn = 8
	)
	ctx := context.Background()

	// --- cluster mode --------------------------------------------------------
	if len(addrs) > 1 {
		cl := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        addrs,
			Password:     password,
			PoolSize:     poolSize,
			MinIdleConns: minIdleConn,
		})
		pingRedis(ctx, cl)
		return &RedisCache{client: cl, ctx: ctx, mode: "cluster"}
	}

	// --- single‑instance mode -------------------------------------------------
	single := redis.NewClient(&redis.Options{
		Addr:         addrs[0],
		Password:     password,
		DB:           0,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConn,
	})
	pingRedis(ctx, single)
	return &RedisCache{client: single, ctx: ctx, mode: "single"}
}

// pingRedis logs whether the connection is alive.
func pingRedis(ctx context.Context, c redis.Cmdable) {
	if err := c.Ping(ctx).Err(); err != nil {
		logger.Log.Error().Err(err).Msg("redis ping failed")
	} else {
		logger.Log.Info().Msg("redis ping succeeded")
	}
}

// ---------------------------------------------------------------------------
// Primitive helpers
// ---------------------------------------------------------------------------

// SetJSONPipeline stores a value (marshalled to JSON) and its TTL in a
// single round‑trip using a pipeline.
func (r *RedisCache) SetJSONPipeline(key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	pipe := r.client.Pipeline()
	pipe.Set(r.ctx, key, data, ttl) // SET already accepts TTL, but we add EXPIRE
	if ttl > 0 {
		pipe.Expire(r.ctx, key, ttl)
	}
	_, err = pipe.Exec(r.ctx)
	return err
}

// AddToSetBulk pushes many members into a set and optionally sets its TTL,
// again in a single round‑trip.
func (r *RedisCache) AddToSetBulk(setKey string, members []string, ttl time.Duration) error {
	if len(members) == 0 {
		return nil
	}

	// Build the argument slice []interface{} for SADD.
	args := make([]interface{}, len(members))
	for i, m := range members {
		args[i] = m
	}

	pipe := r.client.Pipeline()
	pipe.SAdd(r.ctx, setKey, args...)
	if ttl > 0 {
		pipe.Expire(r.ctx, setKey, ttl)
	}
	_, err := pipe.Exec(r.ctx)
	return err
}

// Get returns the raw string value stored under key.  It is used by the
// QueryTxFromCache path.
func (r *RedisCache) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}
