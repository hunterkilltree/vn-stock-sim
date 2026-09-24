package crypto

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

const (
	liveCacheTTL = 10 * time.Second
	liveDownFor  = 30 * time.Second
)

// LiveProvider is the crypto twin of market.LiveProvider: try the live
// source, fall back per call to the mock generator on any error or empty
// answer, cache successes briefly, and skip the live source for a while
// after a connection/HTTP failure. Every answer says which source it
// came from. live may be nil (MARKET_DATA_SOURCE=mock).
type LiveProvider struct {
	live DataProvider
	mock MockProvider

	mu        sync.Mutex
	cache     map[string]cacheEntry
	downUntil time.Time
}

type cacheEntry struct {
	value   any
	expires time.Time
}

func NewLiveProvider(live DataProvider) *LiveProvider {
	return &LiveProvider{live: live, cache: map[string]cacheEntry{}}
}

func (l *LiveProvider) useLive(p Pair) bool {
	if l.live == nil || !p.listed {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return time.Now().After(l.downUntil)
}

func (l *LiveProvider) failed(err error) {
	if err == nil || errors.Is(err, ErrNoData) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if time.Now().Before(l.downUntil) {
		return
	}
	l.downUntil = time.Now().Add(liveDownFor)
	log.Printf("crypto: Binance unreachable (%v), serving mock data for %s", err, liveDownFor)
}

func (l *LiveProvider) cached(key string) (any, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.cache[key]
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.value, true
}

func (l *LiveProvider) store(key string, v any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache[key] = cacheEntry{value: v, expires: time.Now().Add(liveCacheTTL)}
}

func (l *LiveProvider) Ticker(p Pair) Ticker {
	key := "t|" + p.Symbol
	if v, ok := l.cached(key); ok {
		return v.(Ticker)
	}
	if l.useLive(p) {
		t, err := l.live.Ticker(p)
		if err == nil {
			l.store(key, t)
			return t
		}
		l.failed(err)
	}
	t, err := l.mock.Ticker(p)
	if err != nil {
		return Ticker{Source: "mock"}
	}
	return t
}

type barsResult struct {
	bars   []market.Bar
	source string
}

func (l *LiveProvider) Klines(p Pair, interval string, from, to int64) ([]market.Bar, string) {
	key := fmt.Sprintf("k|%s|%s|%d|%d", p.Symbol, interval, from, to)
	if v, ok := l.cached(key); ok {
		r := v.(barsResult)
		return r.bars, r.source
	}
	if l.useLive(p) {
		bars, err := l.live.Klines(p, interval, from, to)
		if err == nil && len(bars) > 0 {
			l.store(key, barsResult{bars, "binance"})
			return bars, "binance"
		}
		l.failed(err)
	}
	bars, _ := l.mock.Klines(p, interval, from, to)
	return bars, "mock"
}

func (l *LiveProvider) Depth(p Pair, limit int) ([]DepthLevel, []DepthLevel, string) {
	if l.useLive(p) {
		bids, asks, err := l.live.Depth(p, limit)
		if err == nil {
			return bids, asks, "binance"
		}
		l.failed(err)
	}
	bids, asks, _ := l.mock.Depth(p, limit)
	return bids, asks, "mock"
}
