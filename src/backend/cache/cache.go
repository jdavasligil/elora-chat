package cache

import (
	"time"
)

type CacheStrategy byte

const (
	StratRedis CacheStrategy = iota
	StratDB
	StratLocal
)

type CacheMessage struct {
	ID    string
	Value any
}

// Cache supports key/value storage and append-only logging and streaming
type Cache interface {
	// Keystore operations

	Get(key string) (string, error)
	Set(key string, value string, expiration time.Duration) error
	Del(keys ...string) error

	// Stream operations

	XAdd(stream string, values map[string]any, maxlen int64) (string, error)
	XGetNew(stream string, key string) ([]CacheMessage, error)
	XGetLastN(stream string, key string, count int64) ([]CacheMessage, error)

	// Connection

	Ping() (string, error)
}
