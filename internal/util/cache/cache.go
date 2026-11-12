// Package cache provides a tiny in-memory, TTL based cache. It is used to
// avoid re-querying every provider for identical search requests.
package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value V
	exp   time.Time
}

// Cache is a generic, concurrency-safe cache with per-entry TTL.
type Cache[V any] struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]entry[V]
}

// New creates a cache. A ttl <= 0 disables expiry (entries live until evicted).
func New[V any](ttl time.Duration) *Cache[V] {
	return &Cache[V]{ttl: ttl, items: make(map[string]entry[V])}
}

// Get returns the cached value if present and not expired.
func (c *Cache[V]) Get(key string) (V, bool) {
	var zero V
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return zero, false
	}
	if !e.exp.IsZero() && time.Now().After(e.exp) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return zero, false
	}
	return e.value, true
}

// Set stores a value under key.
func (c *Cache[V]) Set(key string, value V) {
	var exp time.Time
	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl)
	}
	c.mu.Lock()
	c.items[key] = entry[V]{value: value, exp: exp}
	c.mu.Unlock()
}

// Len returns the number of entries currently held.
func (c *Cache[V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
