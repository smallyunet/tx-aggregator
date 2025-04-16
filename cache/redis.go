package cache

import (
	"context"
	"encoding/json"
	"time"
	"tx-aggregator/logger"

	"github.com/redis/go-redis/v9"
)

// RedisCache provides a simple wrapper around a Redis client with context management.
// It supports both single instance and cluster mode Redis connections.
type RedisCache struct {
	singleClient *redis.Client        // Redis client instance for single node operations
	client       *redis.ClusterClient // Redis client instance for cluster operations
	ctx          context.Context      // Context for Redis operations
}

// NewRedisCache initializes a new Redis client with the given address and password.
// It creates a single instance Redis connection.
// Parameters:
//   - addr: Redis server address in format "host:port"
//   - password: Password for Redis authentication
//
// Returns:
//   - *RedisCache: Initialized Redis cache instance
func NewRedisCache(addr, password string) *RedisCache {
	logger.Log.Info().
		Str("addr", addr).
		Msg("Initializing new Redis single instance client")

	return &RedisCache{
		singleClient: redis.NewClient(&redis.Options{
			Addr:     addr,     // Redis server address
			Password: password, // Password for Redis authentication
			DB:       0,        // Use default DB
		}),
		ctx: context.Background(),
	}
}

// NewRedisClusterClient initializes a new Redis cluster client and verifies the connection.
// It creates both cluster and single instance clients for flexibility.
// Parameters:
//   - addrs: List of Redis cluster node addresses
//   - password: Password for Redis authentication
//
// Returns:
//   - *RedisCache: Initialized Redis cache instance with cluster support
func NewRedisClusterClient(addrs []string, password string) *RedisCache {
	ctx := context.Background()
	logger.Log.Info().
		Strs("addrs", addrs).
		Msg("Initializing new Redis cluster client")

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    addrs,    // A list of cluster node addresses
		Password: password, // Password for Redis authentication
	})

	// Try to ping the cluster to verify the connection
	if err := client.Ping(ctx).Err(); err != nil {
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

	sigleClient := redis.NewClient(&redis.Options{
		Addr:     addrs[0], // Use the first address for single client
		Password: password,
	})

	// Try to ping the single client to verify the connection
	if err := sigleClient.Ping(ctx).Err(); err != nil {
		logger.Log.Error().
			Stack().
			Err(err).
			Strs("addrs", addrs).
			Msg("Redis single client initialization failed")
	} else {
		logger.Log.Info().
			Str("addr", addrs[0]).
			Msg("Redis single client initialized successfully")
	}

	return &RedisCache{
		singleClient: sigleClient,
		client:       client,
		ctx:          ctx,
	}
}

// Get retrieves a value associated with the given key as a string.
// Parameters:
//   - key: The key to retrieve
//
// Returns:
//   - string: The value associated with the key
//   - error: Any error that occurred during the operation
func (r *RedisCache) Get(key string) (string, error) {
	logger.Log.Debug().
		Str("key", key).
		Msg("Retrieving value from Redis")

	value, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		logger.Log.Error().
			Err(err).
			Str("key", key).
			Msg("Failed to retrieve value from Redis")
		return "", err
	}

	logger.Log.Debug().
		Str("key", key).
		Msg("Successfully retrieved value from Redis")
	return value, nil
}

// Set stores a value under the given key with an expiration time.
// The value is marshaled into JSON format before storing.
// Parameters:
//   - key: The key to store the value under
//   - value: The value to store (will be JSON marshaled)
//   - expiration: The duration after which the key will expire
//
// Returns:
//   - error: Any error that occurred during the operation
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
// Parameters:
//   - key: The key to delete
//
// Returns:
//   - error: Any error that occurred during the operation
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

// AddToSet adds a member to a Redis set identified by setKey.
// If expiration is greater than 0, it sets the TTL on the setKey.
// Parameters:
//   - setKey: The key identifying the set
//   - member: The member to add to the set
//   - expiration: The duration after which the set will expire (0 for no expiration)
//
// Returns:
//   - error: Any error that occurred during the operation
func (r *RedisCache) AddToSet(setKey string, member string, expiration time.Duration) error {
	logger.Log.Debug().
		Str("setKey", setKey).
		Str("member", member).
		Dur("expiration", expiration).
		Msg("Adding member to Redis set")

	// Add member to the set
	if err := r.client.SAdd(r.ctx, setKey, member).Err(); err != nil {
		logger.Log.Error().
			Err(err).
			Str("setKey", setKey).
			Str("member", member).
			Msg("Failed to add member to Redis set")
		return err
	}

	// Set expiration on the key if needed
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
// Parameters:
//   - setKey: The key identifying the set
//
// Returns:
//   - []string: List of members in the set
//   - error: Any error that occurred during the operation
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
