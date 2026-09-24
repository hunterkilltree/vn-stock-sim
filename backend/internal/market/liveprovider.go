package market

import (
	"errors"
	"log"
	"strconv"
	"sync"
	"time"
)

// liveCacheTTL is short on purpose: long enough to absorb one page
// load's fan-out of several GetBars/GetIndicator/GetIndex calls that
// often share the same underlying series, short enough that "now" stays
// genuinely fresh -- see phase-vci-market-data.md decision 5.
const liveCacheTTL = 5 * time.Second

// LiveProvider composes a "live" adapter (VCIProvider) with MockProvider
// as a fallback: if the live call fails or comes back empty (a real
// possibility for an undocumented, unofficial upstream -- see
// vciprovider.go), that one call degrades to the deterministic mock
// generator instead of breaking the page. LiveProvider is itself just
// another MarketDataProvider, so main.go still wires exactly one
// implementation into market.NewService either way -- composing
// providers behind the port, not branching inside callers (phase-vci-
// market-data.md decision 4).
type LiveProvider struct {
	live *VCIProvider
	mock *MockProvider

	mu        sync.Mutex
	barsCache map[string]barsCacheEntry
	downUntil time.Time
	idxCache  map[string]idxCacheEntry
}

// liveDownFor is how long the provider skips VCI after a connection or
// HTTP failure. Without it, an unreachable VCI makes every one of the
// heatmap's ~36 per-symbol requests wait on a failing call before
// falling back (phase-i.md decision 14).
const liveDownFor = 30 * time.Second

func (p *LiveProvider) liveDown() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return time.Now().Before(p.downUntil)
}

func (p *LiveProvider) markDown(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Now().Before(p.downUntil) {
		return
	}
	p.downUntil = time.Now().Add(liveDownFor)
	log.Printf("market: VCI unreachable (%v), serving mock data for %s", err, liveDownFor)
}

type barsCacheEntry struct {
	bars    []Bar
	expires time.Time
}

type idxCacheEntry struct {
	snapshot IndexSnapshot
	expires  time.Time
}

func NewLiveProvider(live *VCIProvider, mock *MockProvider) *LiveProvider {
	return &LiveProvider{
		live:      live,
		mock:      mock,
		barsCache: make(map[string]barsCacheEntry),
		idxCache:  make(map[string]idxCacheEntry),
	}
}

func (p *LiveProvider) GetBars(sym, resolution string, from, to int64) []Bar {
	key := sym + "|" + resolution + "|" + strconv.FormatInt(from, 10) + "|" + strconv.FormatInt(to, 10)

	p.mu.Lock()
	if entry, ok := p.barsCache[key]; ok && time.Now().Before(entry.expires) {
		p.mu.Unlock()
		return entry.bars
	}
	p.mu.Unlock()

	if p.liveDown() {
		return p.mock.GetBars(sym, resolution, from, to)
	}
	bars, err := p.live.getBars(sym, resolution, from, to)
	if err != nil && !errors.Is(err, errVCINoData) {
		p.markDown(err)
	}
	if len(bars) == 0 {
		return p.mock.GetBars(sym, resolution, from, to)
	}

	p.mu.Lock()
	p.barsCache[key] = barsCacheEntry{bars: bars, expires: time.Now().Add(liveCacheTTL)}
	p.mu.Unlock()
	return bars
}

func (p *LiveProvider) GetIndex(name string) IndexSnapshot {
	p.mu.Lock()
	if entry, ok := p.idxCache[name]; ok && time.Now().Before(entry.expires) {
		p.mu.Unlock()
		return entry.snapshot
	}
	p.mu.Unlock()

	if p.liveDown() {
		return p.mock.GetIndex(name)
	}
	snapshot := p.live.GetIndex(name)
	if len(snapshot.Sparkline) == 0 {
		log.Printf("market: VCI returned no data for index %s, falling back to mock data", name)
		return p.mock.GetIndex(name)
	}

	p.mu.Lock()
	p.idxCache[name] = idxCacheEntry{snapshot: snapshot, expires: time.Now().Add(liveCacheTTL)}
	p.mu.Unlock()
	return snapshot
}

// GetOrderBook is always synthetic (phase-vci-market-data.md decision
// 2) -- no live call to fall back from, so this goes straight to mock.
func (p *LiveProvider) GetOrderBook(sym string, lastPrice float64) (bids, asks []PriceLevel) {
	return p.mock.GetOrderBook(sym, lastPrice)
}
