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
	if req.Rule.Type != "ema_crossover" {
		return Backtest{}, ErrUnsupportedRule
	}
	from, to, err := parseDateRange(req.From, req.To)
	if err != nil {
		return Backtest{}, err
	}
	bars := s.bars.GetBars(req.Symbol, "1D", from, to)

	fast := req.Rule.Params["fast"]
	slow := req.Rule.Params["slow"]
	if fast <= 0 {
		fast = 20
	}
	if slow <= 0 {
		slow = 50
	}

	result := runEMACrossover(bars, fast, slow, req.StartingCapital)
	bt := Backtest{
		Symbol:        req.Symbol,
		Status:        "completed",
		CreatedAt:     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		FinalCapital:  round2(result.finalCapital),
		ReturnPercent: round2(result.returnPercent),
		TotalTrades:   result.totalTrades,
		WinRate:       round2(result.winRate),
	}
	return s.store.Append(userID, bt), nil
}

func (s *Service) List(userID string) []Backtest {
	return s.store.List(userID)
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
