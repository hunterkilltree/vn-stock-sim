package market

import (
	"sync"
	"time"
)

// swrCache is a small stale-while-revalidate cache with per-key request
// coalescing. A fresh entry is returned as is; a stale one (past its fresh
// window but within its stale window) is returned immediately while one
// background reload refreshes it; a missing or fully expired one blocks on
// a single shared load. Concurrent loads of the same key always collapse
// into one call, so a page's parallel requests share one upstream fetch
// instead of each making their own.
type swrCache[V any] struct {
	mu       sync.Mutex
	entries  map[string]swrEntry[V]
	inflight map[string]*swrCall[V]
	maxSize  int
}

type swrEntry[V any] struct {
	val   V
	fresh time.Time // served as is until here
	stale time.Time // served while a background reload runs until here
}

type swrCall[V any] struct {
	done chan struct{}
	val  V
	ok   bool
}

func newSWRCache[V any](maxSize int) *swrCache[V] {
	return &swrCache[V]{
		entries:  make(map[string]swrEntry[V]),
		inflight: make(map[string]*swrCall[V]),
		maxSize:  maxSize,
	}
}

// get returns the cached value for key, loading it when needed. load
// returns ok=false when the source has nothing usable, which is never
// cached and leaves any existing entry untouched. ttl returns the fresh and
// stale windows to apply to a newly loaded value.
func (c *swrCache[V]) get(key string, ttl func() (fresh, stale time.Duration), load func() (V, bool)) (V, bool) {
	now := time.Now()
	c.mu.Lock()
	e, ok := c.entries[key]
	c.mu.Unlock()

	if ok && now.Before(e.fresh) {
		return e.val, true
	}
	if ok && now.Before(e.stale) {
		go c.reload(key, ttl, load)
		return e.val, true
	}
	return c.reload(key, ttl, load)
}

// reload runs load for key unless one is already in flight, in which case
// it waits for that one. It always bypasses the freshness check, so it
// doubles as the forced refresh used by background refreshers.
func (c *swrCache[V]) reload(key string, ttl func() (fresh, stale time.Duration), load func() (V, bool)) (V, bool) {
	c.mu.Lock()
	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		<-call.done
		return call.val, call.ok
	}
	call := &swrCall[V]{done: make(chan struct{})}
	c.inflight[key] = call
	c.mu.Unlock()

	call.val, call.ok = load()

	c.mu.Lock()
	if call.ok {
		fresh, stale := ttl()
		now := time.Now()
		c.entries[key] = swrEntry[V]{val: call.val, fresh: now.Add(fresh), stale: now.Add(stale)}
		c.pruneLocked(now)
	}
	delete(c.inflight, key)
	c.mu.Unlock()
	close(call.done)

	if !call.ok {
		// Load failed: fall back to whatever stale value is still usable
		// rather than reporting nothing.
		c.mu.Lock()
		e, ok := c.entries[key]
		c.mu.Unlock()
		if ok && time.Now().Before(e.stale) {
			return e.val, true
		}
	}
	return call.val, call.ok
}

// pruneLocked drops fully expired entries once the cache outgrows maxSize,
// so varied request windows can't grow it without bound.
func (c *swrCache[V]) pruneLocked(now time.Time) {
	if len(c.entries) <= c.maxSize {
		return
	}
	for k, e := range c.entries {
		if !now.Before(e.stale) {
			delete(c.entries, k)
		}
	}
}
