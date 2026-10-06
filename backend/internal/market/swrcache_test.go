package market

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func ttl(fresh, stale time.Duration) func() (time.Duration, time.Duration) {
	return func() (time.Duration, time.Duration) { return fresh, stale }
}

func TestSWRCacheCoalescesConcurrentLoads(t *testing.T) {
	c := newSWRCache[int](10)
	var calls atomic.Int32
	load := func() (int, bool) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return 7, true
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if v, ok := c.get("k", ttl(time.Minute, time.Hour), load); !ok || v != 7 {
				t.Errorf("got %d,%v", v, ok)
			}
		}()
	}
	wg.Wait()
	if n := calls.Load(); n != 1 {
		t.Fatalf("load ran %d times, want 1", n)
	}
	c.get("k", ttl(time.Minute, time.Hour), load)
	if calls.Load() != 1 {
		t.Fatal("fresh hit reloaded")
	}
}

func TestSWRCacheServesStaleAndRefreshes(t *testing.T) {
	c := newSWRCache[int](10)
	n := 0
	load := func() (int, bool) { n++; return n, true }
	c.get("k", ttl(time.Millisecond, time.Hour), load)
	time.Sleep(5 * time.Millisecond)
	if v, _ := c.get("k", ttl(time.Millisecond, time.Hour), load); v != 1 {
		t.Fatalf("stale read = %d, want old value 1", v)
	}
	time.Sleep(20 * time.Millisecond) // let the background reload land
	if v, _ := c.get("k", ttl(time.Minute, time.Hour), load); v != 2 {
		t.Fatalf("after refresh = %d, want 2", v)
	}
}

func TestSWRCacheFailedLoadKeepsStale(t *testing.T) {
	c := newSWRCache[int](10)
	c.get("k", ttl(time.Millisecond, time.Hour), func() (int, bool) { return 1, true })
	time.Sleep(5 * time.Millisecond)
	if v, ok := c.reload("k", ttl(time.Minute, time.Hour), func() (int, bool) { return 0, false }); !ok || v != 1 {
		t.Fatalf("got %d,%v, want stale 1", v, ok)
	}
	if _, ok := c.get("missing", ttl(time.Minute, time.Hour), func() (int, bool) { return 0, false }); ok {
		t.Fatal("failed load of a missing key reported ok")
	}
}

func TestBarsWindowLatestSharesKey(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	k1, from, to := barsWindow("VNM", "1D", now.Unix()-864000, now.Unix(), now)
	k2, _, _ := barsWindow("VNM", "1D", now.Unix()+3-864000, now.Unix()+3, now.Add(3*time.Second))
	if k1 != k2 {
		t.Fatalf("latest keys differ: %q vs %q", k1, k2)
	}
	if to != now.Unix() || from > now.Unix()-864000 {
		t.Fatalf("fetch window %d..%d does not cover request", from, to)
	}
	hist, _, _ := barsWindow("VNM", "1D", 1000, 5000, now)
	if hist != "VNM|1D|1000|5000" {
		t.Fatalf("historical key = %q", hist)
	}
}

func TestTrimFrom(t *testing.T) {
	bars := []Bar{{Time: 1}, {Time: 2}, {Time: 3}}
	if got := trimFrom(bars, 2); len(got) != 2 || got[0].Time != 2 {
		t.Fatalf("got %v", got)
	}
}
