package backtest

import (
	"errors"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/market"
)

var ErrUnsupportedRule = errors.New("unsupported rule type")
var ErrNotFound = errors.New("backtest not found")

type BarsPort interface {
	GetBars(sym, resolution string, from, to int64) []market.Bar
}

type Service struct {
	store *MemoryStore
	bars  BarsPort
}

func NewService(store *MemoryStore, bars BarsPort) *Service {
	return &Service{store: store, bars: bars}
}

// Create runs the requested rule synchronously (see store.go's note on the
// deferred worker pool) and stores the completed result.
func (s *Service) Create(userID string, req createRequest) (Backtest, error) {
	from, to, err := parseDateRange(req.From, req.To)
	if err != nil {
		return Backtest{}, err
	}
	bars := s.bars.GetBars(req.Symbol, "1D", from, to)

	var result runResult
	switch req.Rule.Type {
	case "ema_crossover":
		fast := paramOr(req.Rule.Params, "fast", 20)
		slow := paramOr(req.Rule.Params, "slow", 50)
		result = runEMACrossover(bars, fast, slow, req.StartingCapital)
	case "rsi_reversion":
		result = runRSIReversion(bars, rsiReversionParams{
			period:          paramOr(req.Rule.Params, "period", 14),
			entry:           float64(paramOr(req.Rule.Params, "entry", 35)),
			exit:            float64(paramOr(req.Rule.Params, "exit", 70)),
			stopLossPercent: float64(paramOr(req.Rule.Params, "stopLossPercent", 0)),
			trendSMA:        paramOr(req.Rule.Params, "trendSma", 0),
		}, req.StartingCapital)
	default:
		return Backtest{}, ErrUnsupportedRule
	}

	bt := Backtest{
		Symbol:             req.Symbol,
		Status:             "completed",
		CreatedAt:          time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		FinalCapital:       round2(result.finalCapital),
		ReturnPercent:      round2(result.returnPercent),
		TotalTrades:        result.totalTrades,
		WinRate:            round2(result.winRate),
		RuleType:           req.Rule.Type,
		MaxDrawdownPercent: round2(result.maxDrawdownPercent),
		ProfitFactor:       round2(result.profitFactor),

		From:                   req.From,
		To:                     req.To,
		StartingCapital:        req.StartingCapital,
		Params:                 req.Rule.Params,
		BenchmarkReturnPercent: round2(result.benchmarkReturnPercent),
		Equity:                 result.equity,
		Trades:                 result.trades,
	}
	return s.store.Append(userID, bt), nil
}

// paramOr reads an int rule parameter, falling back to def when it's
// missing or not positive.
func paramOr(params map[string]int, key string, def int) int {
	if v, ok := params[key]; ok && v > 0 {
		return v
	}
	return def
}

// List returns summaries: the per-bar equity and trade list stay on the
// single-backtest endpoints, so a long history doesn't bloat the list.
func (s *Service) List(userID string) []Backtest {
	out := s.store.List(userID)
	for i := range out {
		out[i].Equity = nil
		out[i].Trades = nil
	}
	return out
}

func (s *Service) Get(userID, id string) (Backtest, error) {
	bt, ok := s.store.ByID(userID, id)
	if !ok {
		return Backtest{}, ErrNotFound
	}
	return bt, nil
}

func parseDateRange(from, to string) (int64, int64, error) {
	f, err := time.Parse("2006-01-02", from)
	if err != nil {
		return 0, 0, err
	}
	t, err := time.Parse("2006-01-02", to)
	if err != nil {
		return 0, 0, err
	}
	return f.Unix(), t.Unix(), nil
}

func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}
