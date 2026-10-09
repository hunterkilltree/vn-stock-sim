package rating

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/valuation"
)

var ErrSymbolNotFound = errors.New("symbol not found")

// lookbackSeconds: ~8 years of daily bars, enough for SMA 200, the
// 3-year hold history and VCI's 2000-bar countBack cap
// (phase-holding-horizon.md decision 2).
const lookbackSeconds = 8 * 365 * 24 * 60 * 60

// indexName is VN-Index as market.GetBars knows it (VCI maps it to
// VNINDEX; the mock generates a series for it).
const indexName = "VN-Index"

// sectorTTL: the seeded universe is static, but P/E moves with price.
const sectorTTL = time.Hour

type SymbolPort interface {
	Detail(sym string) (symbol.Detail, bool)
	Search(query, exchange string, page, pageSize int) ([]symbol.Symbol, int)
}

type MarketPort interface {
	GetBars(sym, resolution string, from, to int64) []market.Bar
}

// Service rates a symbol Strong Buy .. Strong Sell from fixed rules over
// its daily bars and fundamentals (rules.go), plus the holding-horizon
// verdicts (horizon.go) and today's valuation buy zone -- deterministic,
// no LLM, same reasoning as internal/insight.
type Service struct {
	symbols SymbolPort
	market  MarketPort

	mu         sync.Mutex
	sectorPE   map[string]float64
	sectorTime time.Time
}

func NewService(symbols SymbolPort, market MarketPort) *Service {
	return &Service{symbols: symbols, market: market}
}

func (s *Service) Rate(sym string) (Result, error) {
	detail, ok := s.symbols.Detail(sym)
	if !ok {
		return Result{}, ErrSymbolNotFound
	}
	now := time.Now()
	from, to := now.Unix()-lookbackSeconds, now.Unix()
	bars := s.market.GetBars(detail.Symbol.Symbol, "1D", from, to)
	res := Compute(detail, bars)
	res.GeneratedAt = now.UTC().Format(time.RFC3339)

	if detail.Exchange != symbol.ExchangeCrypto && len(bars) > 0 {
		res.Horizons = ComputeHorizons(HorizonInput{
			Detail:         detail,
			Bars:           bars,
			IndexBars:      s.market.GetBars(indexName, "1D", from, to),
			SectorMedianPE: s.sectorMedianPE(detail.Sector),
		})
		if detail.EPS != 0 {
			v := valuation.Summarize(fundamentals(detail, bars[len(bars)-1].Close), valuation.DefaultParams(),
				func(p float64) float64 { return symbol.FloorToTick(p, detail.Exchange) })
			res.Valuation = &v
		}
	}
	return res, nil
}

// fundamentals maps today's snapshot onto valuation's inputs. Growth and
// the historical P/E and yield series don't exist yet, so the methods
// needing them report "thiếu dữ liệu" (phase-valuation.md decision 3).
func fundamentals(d symbol.Detail, price float64) valuation.Fundamentals {
	return valuation.Fundamentals{
		Price:   price,
		EPS:     d.EPS,
		BVPS:    d.BookValuePerShare,
		ROE:     d.ROE,
		CashDPS: price * d.DividendYield / 100,
	}
}

// sectorMedianPE is the median P/E of profitable stocks in the sector,
// recomputed at most once per sectorTTL.
func (s *Service) sectorMedianPE(sector string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sectorPE == nil || time.Since(s.sectorTime) > sectorTTL {
		all, _ := s.symbols.Search("", "", 1, 10_000)
		bySector := map[string][]float64{}
		for _, sym := range all {
			d, ok := s.symbols.Detail(sym.Symbol)
			if ok && d.EPS > 0 && d.PERatio > 0 {
				bySector[d.Sector] = append(bySector[d.Sector], d.PERatio)
			}
		}
		s.sectorPE = map[string]float64{}
		for sec, pes := range bySector {
			sort.Float64s(pes)
			s.sectorPE[sec] = percentile(pes, 0.5)
		}
		s.sectorTime = time.Now()
	}
	return s.sectorPE[sector]
}
