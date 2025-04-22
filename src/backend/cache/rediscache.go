package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache is a thin layer over redis that implements the Cache interface.
type RedisCache struct {
	ctx    context.Context
	client redis.Client
	lastID []byte
}

func NewRedisCache(ctx context.Context, opt *redis.Options) *RedisCache {
	return &RedisCache{
		ctx:    ctx,
		client: *redis.NewClient(opt),
		lastID: make([]byte, 0, 20), // 20 digits in 64 bit max
	}
}

func (c *RedisCache) Get(key string) (string, error) {
	return c.client.Get(c.ctx, key).Result()
}

func (c *RedisCache) Set(key string, value string, expiration time.Duration) error {
	return c.client.Set(c.ctx, key, value, expiration).Err()
}

func (c *RedisCache) Del(keys ...string) error {
	return c.client.Del(c.ctx, keys...).Err()
}

func (c *RedisCache) XAdd(stream string, value any, maxlen int64) (string, error) {
	return c.client.XAdd(c.ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"message": value},
		MaxLen: maxlen,
		Approx: true,
	}).Result()
}

// Get the newest value from a stream.
func (c *RedisCache) XGetNew(stream string) ([]CacheMessage, error) {
	streams, err := c.client.XRead(c.ctx, &redis.XReadArgs{
		Streams: []string{stream, string(c.lastID)},
		Block:   0,
	}).Result()

	if err != nil {
		return nil, err
	}

	if len(streams) == 0 {
		return nil, errors.New(fmt.Sprintf("Stream: [%s] not found", stream))
	}

	s := streams[0]
	values := make([]CacheMessage, len(s.Messages))
	for i, message := range s.Messages {
		values[i].ID = message.ID
		values[i].Value = message.Values["message"]
	}

	copy(c.lastID, values[len(values)-1].ID)

	return values, nil
}

// Get the last N values from a stream sorted by newest.
func (c *RedisCache) XGetLastN(stream string, count int64) ([]CacheMessage, error) {
	messages, err := c.client.XRevRangeN(c.ctx, stream, "+", "-", count).Result()
	if err != nil {
		return nil, err
	}

	values := make([]CacheMessage, len(messages))

	vIdx := 0
	lastID := "0"
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]

		values[vIdx].ID = message.ID
		values[vIdx].Value = message.Values["message"]

		if message.ID > lastID {
			lastID = message.ID
		}
	}

	copy(c.lastID, lastID)

	return values, nil
}

func (c *RedisCache) Ping() (string, error) {
	return c.client.Ping(c.ctx).Result()
}
