package cache

import (
	"context"
	"encoding/json"
	"time"
	"tx-aggregator/logger"

	"github.com/redis/go-redis/v9"
)

// RedisCache provides a simple wrapper around a Redis client with context management.
// It supports both single instance and cluster mode Redis connections through Cmdable interface.
type RedisCache struct {
	client redis.Cmdable   // Unified Redis client interface (supports both single and cluster)
	ctx    context.Context // Context for Redis operations
	mode   string          // Debug mode: "single" or "cluster"
}

// NewRedisCache initializes a Redis client (single or cluster) based on the number of provided addresses.
// If one address is given, it uses single-instance mode. If multiple, it uses cluster mode.
func NewRedisCache(addrs []string, password string) *RedisCache {
	ctx := context.Background()

	if len(addrs) > 1 {
		// Cluster mode
		logger.Log.Info().
			Strs("addrs", addrs).
			Msg("Initializing Redis cluster client")

		clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    addrs,
			Password: password,
		})

		if err := clusterClient.Ping(ctx).Err(); err != nil {
			logger.Log.Error().
				Stack().
				Err(err).
				Strs("addrs", addrs).
				Msg("Redis cluster client initialization failed")
		} else {
			logger.Log.Info().
				Strs("addrs", addrs).
				Msg("Redis cluster client initialized successfully")
		}

		return &RedisCache{
			client: clusterClient,
			ctx:    ctx,
			mode:   "cluster",
		}
	}

	// Single instance mode
	logger.Log.Info().
		Str("addr", addrs[0]).
		Msg("Initializing Redis single instance client")

	singleClient := redis.NewClient(&redis.Options{
		Addr:     addrs[0],
		Password: password,
		DB:       0,
	})

	if err := singleClient.Ping(ctx).Err(); err != nil {
		logger.Log.Error().
			Stack().
			Err(err).
			Str("addr", addrs[0]).
			Msg("Redis single client initialization failed")
	} else {
		logger.Log.Info().
			Str("addr", addrs[0]).
			Msg("Redis single client initialized successfully")
	}

	return &RedisCache{
		client: singleClient,
		ctx:    ctx,
		mode:   "single",
	}
}

// Get retrieves a value associated with the given key as a string.
func (r *RedisCache) Get(key string) (string, error) {
	logger.Log.Debug().
		Str("key", key).
		Msg("Retrieving value from Redis")

	value, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			logger.Log.Debug().
				Str("key", key).
				Msg("Key not found in Redis")
		} else {
			logger.Log.Error().
				Err(err).
				Str("key", key).
				Msg("Failed to retrieve value from Redis")
		}
		return "", err
	}

	logger.Log.Debug().
		Str("key", key).
		Msg("Successfully retrieved value from Redis")
	return value, nil
}

// Set stores a value under the given key with an expiration time.
func (r *RedisCache) Set(key string, value interface{}, expiration time.Duration) error {
	logger.Log.Debug().
		Str("key", key).
		Dur("expiration", expiration).
		Msg("Setting value in Redis")

	data, err := json.Marshal(value)
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to marshal value to JSON")
		return err
	}

	err = r.client.Set(r.ctx, key, data, expiration).Err()
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to set value in Redis")
		return err
	}

	logger.Log.Debug().
		Str("key", key).
		Msg("Successfully set value in Redis")
	return nil
}

// Delete removes the specified key and its associated value from Redis.
func (r *RedisCache) Delete(key string) error {
	logger.Log.Debug().
		Str("key", key).
		Msg("Deleting key from Redis")

	err := r.client.Del(r.ctx, key).Err()
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to delete key from Redis")
		return err
	}

	logger.Log.Debug().
		Str("key", key).
		Msg("Successfully deleted key from Redis")
	return nil
}

// AddToSet adds a member to a Redis set identified by setKey and optionally sets expiration.
func (r *RedisCache) AddToSet(setKey string, member string, expiration time.Duration) error {
	logger.Log.Debug().
		Str("setKey", setKey).
		Str("member", member).
		Dur("expiration", expiration).
		Msg("Adding member to Redis set")

	if err := r.client.SAdd(r.ctx, setKey, member).Err(); err != nil {
		logger.Log.Error().
			Err(err).
			Str("setKey", setKey).
			Str("member", member).
			Msg("Failed to add member to Redis set")
		return err
	}

	if expiration > 0 {
		if err := r.client.Expire(r.ctx, setKey, expiration).Err(); err != nil {
			logger.Log.Error().
				Err(err).
				Str("setKey", setKey).
				Dur("expiration", expiration).
				Msg("Failed to set expiration on Redis set")
			return err
		}
	}

	logger.Log.Debug().
		Str("setKey", setKey).
		Str("member", member).
		Msg("Successfully added member to Redis set")
	return nil
}

// GetSetMembers retrieves all members of a Redis set under the specified setKey.
func (r *RedisCache) GetSetMembers(setKey string) ([]string, error) {
	logger.Log.Debug().
		Str("setKey", setKey).
		Msg("Retrieving members from Redis set")

	members, err := r.client.SMembers(r.ctx, setKey).Result()
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("setKey", setKey).
			Msg("Failed to retrieve members from Redis set")
		return nil, err
	}

	logger.Log.Debug().
		Str("setKey", setKey).
		Int("memberCount", len(members)).
		Msg("Successfully retrieved members from Redis set")
	return members, nil
}
