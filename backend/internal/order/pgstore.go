package order

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db"
)

// PGStore keeps orders in Postgres (phase-persistence.md), in placement
// order per user; queued orders survive a restart and the matcher picks
// them up again. Errors are logged (decision 9).
type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

const orderCols = `id, portfolio_id, symbol, side, type, quantity, status, price, stop_price, filled_price, fee, filled_at, created_at, triggered_by, reject_reason, stop_loss, target, setup, plan_id, violations`

func scanOrder(row pgx.Row) (Order, error) {
	var o Order
	err := row.Scan(orderDest(&o)...)
	normalize(&o)
	return o, err
}

// normalize matches MemoryStore: no violations is nil, not [].
func normalize(o *Order) {
	if len(o.Violations) == 0 {
		o.Violations = nil
	}
}

// orderDest is orderCols' scan targets, in order.
func orderDest(o *Order) []any {
	return []any{&o.ID, &o.PortfolioID, &o.Symbol, &o.Side, &o.Type, &o.Quantity, &o.Status, &o.Price, &o.StopPrice,
		&o.FilledPrice, &o.Fee, &o.FilledAt, &o.CreatedAt, &o.TriggeredBy, &o.RejectReason,
		&o.StopLoss, &o.Target, &o.Setup, &o.PlanID, &o.Violations}
}

// violationsJSON stores nil as [] so the column's NOT NULL holds.
func violationsJSON(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func (s *PGStore) Append(userID string, o Order) Order {
	ctx := context.Background()
	var n int64
	if err := s.pool.QueryRow(ctx, `SELECT nextval('order_id_seq')`).Scan(&n); err != nil {
		log.Printf("order: append: %v", err)
		return o
	}
	o.ID = db.HexID("ord_", n)
	_, err := s.pool.Exec(ctx, `INSERT INTO orders (user_id, `+orderCols+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`,
		userID, o.ID, o.PortfolioID, o.Symbol, o.Side, o.Type, o.Quantity, o.Status, o.Price, o.StopPrice,
		o.FilledPrice, o.Fee, o.FilledAt, o.CreatedAt, o.TriggeredBy, o.RejectReason,
		o.StopLoss, o.Target, o.Setup, o.PlanID, violationsJSON(o.Violations))
	if err != nil {
		log.Printf("order: append: %v", err)
	}
	return o
}

func (s *PGStore) List(userID string) []Order {
	rows, err := s.pool.Query(context.Background(), `SELECT `+orderCols+` FROM orders WHERE user_id = $1 ORDER BY seq`, userID)
	if err != nil {
		log.Printf("order: list: %v", err)
		return []Order{}
	}
	defer rows.Close()
	out := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			log.Printf("order: list: %v", err)
			break
		}
		out = append(out, o)
	}
	return out
}

func (s *PGStore) ByID(userID, id string) (Order, bool) {
	o, err := scanOrder(s.pool.QueryRow(context.Background(), `SELECT `+orderCols+` FROM orders WHERE user_id = $1 AND id = $2`, userID, id))
	return o, err == nil
}

func (s *PGStore) Queued() []queuedOrder {
	rows, err := s.pool.Query(context.Background(),
		`SELECT user_id, `+orderCols+` FROM orders WHERE status = 'queued' ORDER BY created_at, seq`)
	if err != nil {
		log.Printf("order: queued: %v", err)
		return nil
	}
	defer rows.Close()
	var out []queuedOrder
	for rows.Next() {
		var q queuedOrder
		o := &q.order
		if err := rows.Scan(append([]any{&q.userID}, orderDest(o)...)...); err != nil {
			log.Printf("order: queued: %v", err)
			break
		}
		normalize(o)
		out = append(out, q)
	}
	return out
}

func (s *PGStore) Replace(userID string, o Order) Order {
	_, err := s.pool.Exec(context.Background(), `UPDATE orders SET status = $3, price = $4, stop_price = $5, filled_price = $6, fee = $7,
		filled_at = $8, created_at = $9, triggered_by = $10, reject_reason = $11, plan_id = $12 WHERE user_id = $1 AND id = $2`,
		userID, o.ID, o.Status, o.Price, o.StopPrice, o.FilledPrice, o.Fee, o.FilledAt, o.CreatedAt, o.TriggeredBy, o.RejectReason, o.PlanID)
	if err != nil {
		log.Printf("order: replace: %v", err)
	}
	return o
}

const planCols = `id, portfolio_id, symbol, setup, planned_entry, initial_stop, current_stop, target, opened_at, closed_at, exit_price, r_multiple`

func planDest(p *TradePlan) []any {
	return []any{&p.ID, &p.PortfolioID, &p.Symbol, &p.Setup, &p.PlannedEntry, &p.InitialStop, &p.CurrentStop, &p.Target,
		&p.OpenedAt, &p.ClosedAt, &p.ExitPrice, &p.RMultiple}
}

func (s *PGStore) AppendPlan(userID string, p TradePlan) TradePlan {
	ctx := context.Background()
	var n int64
	if err := s.pool.QueryRow(ctx, `SELECT nextval('trade_plan_id_seq')`).Scan(&n); err != nil {
		log.Printf("order: append plan: %v", err)
		return p
	}
	p.ID = db.HexID("tp_", n)
	_, err := s.pool.Exec(ctx, `INSERT INTO trade_plans (user_id, `+planCols+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		userID, p.ID, p.PortfolioID, p.Symbol, p.Setup, p.PlannedEntry, p.InitialStop, p.CurrentStop, p.Target,
		p.OpenedAt, p.ClosedAt, p.ExitPrice, p.RMultiple)
	if err != nil {
		log.Printf("order: append plan: %v", err)
	}
	return p
}

func (s *PGStore) OpenPlan(portfolioID, sym string) (TradePlan, bool) {
	var p TradePlan
	err := s.pool.QueryRow(context.Background(),
		`SELECT `+planCols+` FROM trade_plans WHERE portfolio_id = $1 AND symbol = $2 AND closed_at = ''`, portfolioID, sym,
	).Scan(planDest(&p)...)
	return p, err == nil
}

// UpdatePlan leaves initial_stop alone: it is written once, on insert.
func (s *PGStore) UpdatePlan(p TradePlan) {
	_, err := s.pool.Exec(context.Background(), `UPDATE trade_plans SET setup = $2, current_stop = $3, target = $4,
		closed_at = $5, exit_price = $6, r_multiple = $7 WHERE id = $1`,
		p.ID, p.Setup, p.CurrentStop, p.Target, p.ClosedAt, p.ExitPrice, p.RMultiple)
	if err != nil {
		log.Printf("order: update plan: %v", err)
	}
}

func (s *PGStore) Plans(userID string) []TradePlan {
	rows, err := s.pool.Query(context.Background(), `SELECT `+planCols+` FROM trade_plans WHERE user_id = $1 ORDER BY seq`, userID)
	if err != nil {
		log.Printf("order: plans: %v", err)
		return []TradePlan{}
	}
	defer rows.Close()
	out := []TradePlan{}
	for rows.Next() {
		var p TradePlan
		if err := rows.Scan(planDest(&p)...); err != nil {
			log.Printf("order: plans: %v", err)
			break
		}
		out = append(out, p)
	}
	return out
}
