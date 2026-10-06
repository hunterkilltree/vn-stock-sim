package market

import (
	"errors"
	"log"
	"strconv"
	"sync"
	"time"
)

// Cache windows, see phase-vci-market-data.md decision 5. Fresh values are
// served as is; stale ones are served instantly while one background
// reload refreshes them (swrcache.go), so a page load almost never waits
// on VCI. Outside session hours nothing changes, so values stay fresh far
// longer.
const (
	liveFreshOpen   = 15 * time.Second
	liveFreshClosed = 5 * time.Minute
	liveStaleFor    = 15 * time.Minute
	liveCacheMax    = 4000

	// A request whose "to" is within this of now is a "latest" request:
	// callers compute to=now themselves, so an exact-window key would
	// change every second and never hit.
	latestSlack = 120 // seconds
	// "Latest" windows are keyed by span rounded up to this, so spans like
	// 10d and 10d+3s share one entry.
	spanBucket = 3600 // seconds
)

var vnLocation = time.FixedZone("ICT", 7*60*60)

// vnSessionOpen reports whether the HOSE/HNX/UPCOM continuous session
// (09:00-15:00 ICT, Mon-Fri) is plausibly running at t. Holidays are not
// modelled: a holiday just gets the short TTL, which is harmless.
func vnSessionOpen(t time.Time) bool {
	t = t.In(vnLocation)
	if wd := t.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	h := t.Hour()
	return h >= 9 && h < 15
}

func liveTTL() (fresh, stale time.Duration) {
	if vnSessionOpen(time.Now()) {
		return liveFreshOpen, liveStaleFor
	}
	return liveFreshClosed, liveStaleFor
}

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
	downUntil time.Time

	barsCache *swrCache[[]Bar]
	idxCache  *swrCache[IndexSnapshot]
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

func NewLiveProvider(live *VCIProvider, mock *MockProvider) *LiveProvider {
	return &LiveProvider{
		live:      live,
		mock:      mock,
		barsCache: newSWRCache[[]Bar](liveCacheMax),
		idxCache:  newSWRCache[IndexSnapshot](liveCacheMax),
	}
}

// barsWindow picks the cache key and the window actually fetched. "Latest"
// requests (to ~ now) share one entry per symbol/resolution/span and fetch
// up to the present; anything else keys on its exact window.
func barsWindow(sym, resolution string, from, to int64, now time.Time) (key string, fetchFrom, fetchTo int64) {
	nowUnix := now.Unix()
	if to <= from || nowUnix-to > latestSlack || to-nowUnix > latestSlack {
		return sym + "|" + resolution + "|" + strconv.FormatInt(from, 10) + "|" + strconv.FormatInt(to, 10), from, to
	}
	span := (to - from + spanBucket - 1) / spanBucket * spanBucket
	return sym + "|" + resolution + "|L|" + strconv.FormatInt(span, 10), nowUnix - span, nowUnix
}

// trimFrom drops bars before from; cached "latest" entries cover a window
// rounded up past what any one caller asked for.
func trimFrom(bars []Bar, from int64) []Bar {
	i := 0
	for i < len(bars) && bars[i].Time < from {
		i++
	}
	return bars[i:]
}

func (p *LiveProvider) GetBars(sym, resolution string, from, to int64) []Bar {
	key, fetchFrom, fetchTo := barsWindow(sym, resolution, from, to, time.Now())
	bars, ok := p.barsCache.get(key, liveTTL, func() ([]Bar, bool) {
		if p.liveDown() {
			return nil, false
		}
		bars, err := p.live.getBars(sym, resolution, fetchFrom, fetchTo)
		if err != nil && !errors.Is(err, errVCINoData) {
			p.markDown(err)
		}
		return bars, len(bars) > 0
	})
	if !ok {
		return p.mock.GetBars(sym, resolution, from, to)
	}
	return trimFrom(bars, from)
}

func (p *LiveProvider) GetIndex(name string) IndexSnapshot {
	snapshot, ok := p.idxCache.get(name, liveTTL, func() (IndexSnapshot, bool) {
		if p.liveDown() {
			return IndexSnapshot{}, false
		}
		snapshot := p.live.GetIndex(name)
		if len(snapshot.Sparkline) == 0 {
			log.Printf("market: VCI returned no data for index %s, falling back to mock data", name)
			return IndexSnapshot{}, false
		}
		return snapshot, true
	})
	if !ok {
		return p.mock.GetIndex(name)
	}
	return snapshot
}

// GetOrderBook is always synthetic (phase-vci-market-data.md decision
// 2) -- no live call to fall back from, so this goes straight to mock.
func (p *LiveProvider) GetOrderBook(sym string, lastPrice float64) (bids, asks []PriceLevel) {
	return p.mock.GetOrderBook(sym, lastPrice)
}
