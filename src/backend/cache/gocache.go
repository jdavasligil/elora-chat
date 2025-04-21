package cache

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

type stream struct {

	// Heap with subarray?
}

type goCacheValue struct {
	Value     any
	ExpiresAt time.Time
}

func (cv goCacheValue) isExpired() bool {
	return time.Now().After(cv.ExpiresAt)
}

type GoCacheOptions struct {
	SweepPeriod time.Duration
}

// GoCache is a native in memory cache.
type GoCache struct {
	// Key value store optimized for one time write, many time reads.
	store   sync.Map
	expired []string

	// map[string]RingBuffer
	streams sync.Map
	nextID  uint64

	nextSweep   time.Time
	SweepPeriod time.Duration
}

func NewGoCache(opt *GoCacheOptions) *GoCache {
	if opt == nil {
		return &GoCache{
			SweepPeriod: time.Minute,
			nextSweep:   time.Now().Add(time.Minute),
			expired:     make([]string, 0),
		}
	}
	return &GoCache{
		SweepPeriod: opt.SweepPeriod,
		nextSweep:   time.Now().Add(opt.SweepPeriod),
		expired:     make([]string, 0),
	}
}

func (c *GoCache) Get(key string) (string, error) {
	var err error

	v, ok := c.store.Load(key)
	if !ok {
		err = errors.New(fmt.Sprintf("cache: key %s not found.\n", key))
	}

	value := v.(goCacheValue)

	if value.isExpired() {
		err = errors.New(fmt.Sprintf("cache: value expired at %s\n", value.ExpiresAt.String()))
		return "", err
	}

	return v.(string), err
}

// Zero expiration means the key has no expiration time.
func (c *GoCache) Set(key string, value string, expiration time.Duration) error {
	c.store.Store(key, goCacheValue{value, time.Now().Add(expiration)})
	return nil
}

func (c *GoCache) Del(keys ...string) error {
	for _, key := range keys {
		c.store.Delete(key)
	}
	return nil
}

// Note that ring buffers are int32 limited in maxlen
//
//	_, err = redisClient.XAdd(ctx, &redis.XAddArgs{
//		Stream: "chatMessages",
//		Values: map[string]interface{}{"message": string(modifiedMessage)},
//		MaxLen: 100,
//		Approx: true,
//	}).Result()
//
// REF: https://redis.io/docs/latest/develop/data-types/streams/
// TODO: Implement this shit properly. Do we need ring buffers? If so where?
func (c *GoCache) XAdd(stream string, values map[string]any, maxlen int64) (string, error) {
	if values == nil {
		return "", errors.New("cache: values is a nil map")
	}
	if maxlen > math.MaxInt32 {
		return "", errors.New(fmt.Sprintf("cache: max length of ring buffer is %d\n", math.MaxInt32))
	}
	rb := NewRingBuffer[map[string]any](int(maxlen))
	c.streams.LoadOrStore(stream, rb)

	return stream, nil
}

// TODO:
func (c *GoCache) XGetNew(stream string, key string) ([]CacheMessage, error) {
	return nil, nil
}

// TODO:
func (c *GoCache) XGetLastN(stream string, key string, count int64) ([]CacheMessage, error) {
	return nil, nil
}

// For speed, the expiration cache never gets shrunk.
func (c *GoCache) Clean() {
	c.Del(c.expired...)
	c.expired = c.expired[:0]
}

// Search keys for expired values and mark them for removal.
func (c *GoCache) Sweep() {
	c.store.Range(func(key, value any) bool {
		if value.(goCacheValue).isExpired() {
			c.expired = append(c.expired, key.(string))
		}
		return true
	})
}

// Ping is used to prompt the cache to update.
func (c *GoCache) Ping() (string, error) {
	if time.Now().After(c.nextSweep) {
		c.Sweep()
		c.Clean()
		c.nextSweep = time.Now().Add(c.SweepPeriod)
	}
	return "pong", nil
}
