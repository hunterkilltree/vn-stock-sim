package order

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/portfolio"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/riskrules"
	"github.com/hunterkilltree/vn-stock-sim/backend/internal/symbol"
)

var ErrSymbolNotFound = errors.New("symbol not found")
var ErrInsufficientFunds = portfolio.ErrInsufficientCash
var ErrInsufficientShares = portfolio.ErrInsufficientShares
var ErrNotFound = errors.New("order not found")

type Ledger interface {
	// fee is charged to cash on top of the fill value (phase-k.md decision 1).
	ApplyFill(portfolioID, sym, side string, quantity, price, fee float64) error
}

type QuotePort interface {
	Detail(sym string) (symbol.Detail, bool)
}

var ErrPortfolioNotFound = errors.New("portfolio not found")
var ErrWrongMarket = errors.New("this symbol belongs to the other market than this portfolio")
var ErrInvalidQuantity = errors.New("invalid quantity for this market")
var ErrOCOPrices = errors.New("an OCO order needs both a limit price and a stop price")
var ErrPriceRequired = errors.New("limit and stop orders need a price")
var ErrReplayPortfolio = errors.New("replay portfolios only accept fills from their own replay session")
var ErrOddLot = errors.New("stock buys must be whole lots of 100 shares")

// lotSize is the HOSE board lot. Buys are whole lots (phase-risk-rules.md
// slice 1); sells may be odd, so a position bought before the rule can
// still be closed.
const lotSize = 100

// RuleViolation is a buy the portfolio's rules refuse: an invalid plan in
// any mode, or a blocking finding in strict mode. Findings holds only the
// refusing ones, not the warnings that came with them.
type RuleViolation struct {
	Findings []riskrules.Finding
}

func (e *RuleViolation) Error() string { return "order violates the portfolio's risk rules" }

// PortfolioResolver lets Service resolve a request's optional
// portfolioId to a real portfolio the caller owns -- satisfied by
// *portfolio.Service. See phase-b.md decision 1.
type PortfolioResolver interface {
	DefaultPortfolioID(userID string) string
	GetPortfolio(userID, id string) (portfolio.Portfolio, bool)
	Summary(portfolioID string) portfolio.Summary
	Positions(portfolioID string) []portfolio.Position
	RiskSettings(portfolioID string) riskrules.Settings
}

type Service struct {
	store      Store
	ledger     Ledger
	quotes     QuotePort
	portfolios PortfolioResolver
	// mu serialises fills from the matcher with Create/Cancel, so a
	// cancelled order can't also fill (matcher.go).
	mu sync.Mutex
	// bars and checked are the matcher's bar source and per-order
	// watermark (last bar time examined); nil bars = matching off.
	bars    BarsPort
	checked map[string]int64
}

func NewService(store Store, ledger Ledger, quotes QuotePort, portfolios PortfolioResolver) *Service {
	return &Service{store: store, ledger: ledger, quotes: quotes, portfolios: portfolios}
}

// prepared is a validated request: the resolved portfolio, the quote,
// the fee rate, and (for a buy) the risk assessment and the open plan
// it would add to.
type prepared struct {
	req        createRequest
	pf         portfolio.Portfolio
	detail     symbol.Detail
	feeRate    float64
	assessment riskrules.Assessment
	openPlan   *TradePlan
}

// prepare validates req and runs the risk rules (phase-risk-rules.md
// slice 1). Create books what it returns; Check only reports it.
func (s *Service) prepare(userID string, req createRequest) (prepared, error) {
	detail, ok := s.quotes.Detail(req.Symbol)
	if !ok {
		return prepared{}, ErrSymbolNotFound
	}
	if req.PortfolioID == "" {
		req.PortfolioID = s.portfolios.DefaultPortfolioID(userID)
	}
	pf, ok := s.portfolios.GetPortfolio(userID, req.PortfolioID)
	if !ok {
		return prepared{}, ErrPortfolioNotFound
	}
	if pf.Kind == portfolio.KindReplay {
		return prepared{}, ErrReplayPortfolio
	}
	// A crypto wallet trades only crypto pairs, a stock portfolio only VN
	// tickers; stocks trade whole shares, crypto up to 8 decimals
	// (phase-i.md decisions 5 and 8).
	isCrypto := detail.Exchange == symbol.ExchangeCrypto
	if isCrypto != (pf.Market == "crypto") {
		return prepared{}, ErrWrongMarket
	}
	p := prepared{req: req, pf: pf, detail: detail, feeRate: FeeRate}
	lot := float64(lotSize)
	if isCrypto {
		p.req.Quantity = math.Round(req.Quantity*1e8) / 1e8
		p.feeRate, lot = CryptoFeeRate, 0
	} else if req.Quantity != math.Trunc(req.Quantity) {
		return prepared{}, ErrInvalidQuantity
	} else if req.Side == "buy" && math.Mod(req.Quantity, lotSize) != 0 {
		return prepared{}, ErrOddLot
	}
	if p.req.Quantity <= 0 {
		return prepared{}, ErrInvalidQuantity
	}
	if req.Type == "oco" && (req.Price <= 0 || req.StopPrice <= 0) {
		return prepared{}, ErrOCOPrices
	}
	if (req.Type == "limit" || req.Type == "stop") && req.Price <= 0 {
		return prepared{}, ErrPriceRequired
	}

	if req.Side == "buy" {
		entry := detail.LastPrice
		if req.Type != "market" && req.Price > 0 {
			entry = req.Price
		}
		if plan, ok := s.store.OpenPlan(pf.ID, detail.Symbol.Symbol); ok {
			p.openPlan = &plan
			// Adding to a position keeps its plan's stop and target
			// unless the order brings its own.
			if p.req.StopLoss == 0 {
				p.req.StopLoss = plan.CurrentStop
			}
			if p.req.Target == 0 {
				p.req.Target = plan.Target
			}
		}
		p.assessment = riskrules.Check(riskrules.Intent{
			Side: req.Side, Quantity: p.req.Quantity, Entry: entry,
			Stop: p.req.StopLoss, Target: p.req.Target, FeeRate: p.feeRate, Lot: lot,
		}, s.portfolios.Summary(pf.ID).TotalEquity, s.portfolios.RiskSettings(pf.ID))
	}
	return p, nil
}

// Check is the ticket's dry run: the same validation and rules as
// Create, without placing anything.
func (s *Service) Check(userID string, req createRequest) (riskrules.Assessment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.prepare(userID, req)
	if err != nil {
		return riskrules.Assessment{}, err
	}
	if p.assessment.Findings == nil {
		p.assessment.Findings = []riskrules.Finding{}
	}
	return p.assessment, nil
}

// Create places an order. "market" fills immediately at the current
// quote; limit/stop/OCO fill immediately too when they are already
// marketable at that quote (phase-k.md decision 6), otherwise they -- and
// every ATC order -- are stored as "queued" for the matcher (matcher.go).
// A buy the risk rules block returns *RuleViolation; one they only warn
// about is placed with the findings recorded on it.
func (s *Service) Create(userID string, req createRequest) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.prepare(userID, req)
	if err != nil {
		return Order{}, err
	}
	if p.assessment.Blocked {
		rv := &RuleViolation{}
		for _, f := range p.assessment.Findings {
			if f.Severity != riskrules.Warn {
				rv.Findings = append(rv.Findings, f)
			}
		}
		return Order{}, rv
	}
	req, detail := p.req, p.detail
	now := time.Now()
	o := Order{
		PortfolioID: p.pf.ID,
		Symbol:      detail.Symbol.Symbol,
		Side:        req.Side,
		Type:        req.Type,
		Quantity:    req.Quantity,
		CreatedAt:   formatTime(now),
	}
	if req.Side == "buy" {
		o.StopLoss, o.Target, o.Setup = req.StopLoss, req.Target, req.Setup
		for _, f := range p.assessment.Findings {
			o.Violations = append(o.Violations, f.Code)
		}
	}

	if req.Type != "market" {
		o.Price = req.Price
		o.StopPrice = req.StopPrice
		leg, now := triggeredNow(o, detail.LastPrice)
		if !now || req.Type == "atc" {
			o.Status = "queued"
			return s.store.Append(userID, o), nil
		}
		o.TriggeredBy = leg
	}

	fee := round2(detail.LastPrice * req.Quantity * p.feeRate)
	if err := s.ledger.ApplyFill(p.pf.ID, detail.Symbol.Symbol, req.Side, req.Quantity, detail.LastPrice, fee); err != nil {
		return Order{}, err
	}
	o.Status = "filled"
	o.FilledPrice = detail.LastPrice
	o.Fee = fee
	o.FilledAt = formatTime(now)
	s.trackPlan(userID, &o)
	return s.store.Append(userID, o), nil
}

// trackPlan keeps the portfolio's trade plan in step with a fill (call
// with s.mu held): a buy with a stop opens a plan, or joins the open one
// (whose InitialStop never changes); a sell that empties the position
// closes the plan with its R multiple.
func (s *Service) trackPlan(userID string, o *Order) {
	plan, open := s.store.OpenPlan(o.PortfolioID, o.Symbol)
	switch o.Side {
	case "buy":
		if open {
			o.PlanID = plan.ID
			if o.StopLoss > 0 && o.StopLoss != plan.CurrentStop {
				// Raising or lowering the stop is slice 4 (stop
				// management); the latest stop applies until then.
				plan.CurrentStop = o.StopLoss
			}
			if o.Target > 0 {
				plan.Target = o.Target
			}
			s.store.UpdatePlan(plan)
			return
		}
		if o.StopLoss <= 0 {
			return // no stop, no plan: coach mode let it through
		}
		entry := o.Price
		if o.Type == "market" || entry <= 0 {
			entry = o.FilledPrice
		}
		plan = s.store.AppendPlan(userID, TradePlan{
			PortfolioID: o.PortfolioID, Symbol: o.Symbol, Setup: o.Setup,
			PlannedEntry: entry, InitialStop: o.StopLoss, CurrentStop: o.StopLoss, Target: o.Target,
			OpenedAt: o.FilledAt,
		})
		o.PlanID = plan.ID
	case "sell":
		if !open {
			return
		}
		o.PlanID = plan.ID
		for _, pos := range s.portfolios.Positions(o.PortfolioID) {
			if pos.Symbol == o.Symbol && pos.Quantity > 0 {
				return // partial exit: the plan stays open
			}
		}
		plan.ClosedAt, plan.ExitPrice = o.FilledAt, o.FilledPrice
		if risk := plan.PlannedEntry - plan.InitialStop; risk > 0 {
			plan.RMultiple = math.Round((o.FilledPrice-plan.PlannedEntry)/risk*100) / 100
		}
		s.store.UpdatePlan(plan)
	}
}

// Plans lists the caller's trade plans, oldest first.
func (s *Service) Plans(userID string) []TradePlan {
	return s.store.Plans(userID)
}

func round2(v float64) float64 {
	return float64(int64(v*100)) / 100
}

func (s *Service) List(userID string) []Order {
	return s.store.List(userID)
}

// FilledOrders satisfies portfolio.OrdersPort -- see phase-e.md item 2.
// Only "filled" orders carry a real FilledPrice/FilledAt (market orders,
// and queued ones once the matcher fills them), so those are the only
// ones portfolio.Service's FIFO trade reconstruction needs.
func (s *Service) FilledOrders(userID, portfolioID string) []portfolio.OrderRecord {
	all := s.store.List(userID)
	out := make([]portfolio.OrderRecord, 0, len(all))
	for _, o := range all {
		if o.PortfolioID != portfolioID || o.Status != "filled" {
			continue
		}
		out = append(out, portfolio.OrderRecord{
			Symbol:      o.Symbol,
			Side:        o.Side,
			Quantity:    o.Quantity,
			FilledPrice: o.FilledPrice,
			FilledAt:    o.FilledAt,
		})
	}
	return out
}

func (s *Service) Get(userID, id string) (Order, error) {
	o, ok := s.store.ByID(userID, id)
	if !ok {
		return Order{}, ErrNotFound
	}
	return o, nil
}

// Cancel marks a still-queued (limit) order as cancelled. Filled market
// orders cannot be cancelled.
func (s *Service) Cancel(userID, id string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.store.ByID(userID, id)
	if !ok {
		return Order{}, ErrNotFound
	}
	if o.Status != "queued" {
		return Order{}, errors.New("only queued orders can be cancelled")
	}
	o.Status = "cancelled"
	delete(s.checked, o.ID) // the matcher's watermark for it (nil-map safe)
	return s.store.Replace(userID, o), nil
}
