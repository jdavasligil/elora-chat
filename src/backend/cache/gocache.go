package cache

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"
)

type Stream struct {
	nextID uint64
	buffer *RingBuffer[CacheMessage]
}

type GoCacheValue struct {
	Value     any
	ExpiresAt time.Time
}

func (cv GoCacheValue) isExpired() bool {
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

	// map[string]Stream
	streams sync.Map

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

	value := v.(GoCacheValue)

	if value.isExpired() {
		err = errors.New(fmt.Sprintf("cache: value expired at %s\n", value.ExpiresAt.String()))
		return "", err
	}

	return v.(string), err
}

// Zero expiration means the key has no expiration time.
func (c *GoCache) Set(key string, value string, expiration time.Duration) error {
	t := time.Now()
	if expiration == 0 {
		c.store.Store(key, GoCacheValue{value, t.Add(time.Hour * 8760)})
	} else {
		c.store.Store(key, GoCacheValue{value, t.Add(expiration)})
	}
	return nil
}

func (c *GoCache) Del(keys ...string) error {
	for _, key := range keys {
		c.store.Delete(key)
	}
	return nil
}

func (c *GoCache) XAdd(stream string, value any, maxlen int64) (string, error) {
	if maxlen > math.MaxInt32 {
		return "", errors.New(fmt.Sprintf("cache: max length of ring buffer is %d\n", math.MaxInt32))
	}
	v, ok := c.streams.Load(stream)
	if ok {
		s, _ := v.(Stream)
		s.buffer.Push(CacheMessage{strconv.FormatUint(s.nextID, 36), value})
		s.nextID++
		return stream, nil
	}
	rb := NewRingBuffer[CacheMessage](int(maxlen))
	rb.Push(CacheMessage{strconv.FormatUint(0, 36), value})
	c.streams.Store(stream, Stream{buffer: rb, nextID: 1})
	return stream, nil
}

func (c *GoCache) XGetNew(stream string) ([]CacheMessage, error) {
	v, ok := c.streams.Load(stream)
	if !ok {
		return []CacheMessage{}, errors.New(fmt.Sprintf("cache: stream [%s] does not exist\n", stream))
	}
	s, _ := v.(Stream)
	msgs := make([]CacheMessage, s.buffer.length)
	for range s.buffer.length {
		msgs = append(msgs, s.buffer.Pop())
	}
	return msgs, nil
}

func (c *GoCache) XGetLastN(stream string, count int64) ([]CacheMessage, error) {
	v, ok := c.streams.Load(stream)
	if !ok {
		return nil, errors.New(fmt.Sprintf("cache: stream [%s] does not exist\n", stream))
	}
	s, _ := v.(Stream)
	if s.buffer.IsEmpty() {
		return nil, errors.New("cache: buffer is empty")
	}
	msgs := make([]CacheMessage, 0, min(int(count), s.buffer.length))
	for range cap(msgs) {
		msgs = append(msgs, s.buffer.Pop())
	}
	return msgs, nil
}

// For speed, the expiration cache never gets shrunk.
func (c *GoCache) Clean() {
	c.Del(c.expired...)
	c.expired = c.expired[:0]
}

// Search keys for expired values and mark them for removal.
func (c *GoCache) Sweep() {
	c.store.Range(func(key, value any) bool {
		if value.(GoCacheValue).isExpired() {
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
