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
	Get(key string) (string, error)
	Set(key string, value string, expiration time.Duration) error
	Del(keys ...string) error

	XAdd(stream string, value any, maxlen int64) (string, error)
	XGetNew(stream string) ([]CacheMessage, error)
	XGetLastN(stream string, count int64) ([]CacheMessage, error)

	Ping() (string, error)
}
