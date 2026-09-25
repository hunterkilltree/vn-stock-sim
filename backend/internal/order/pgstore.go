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

const orderCols = `id, portfolio_id, symbol, side, type, quantity, status, price, stop_price, filled_price, fee, filled_at, created_at, triggered_by, reject_reason`

func scanOrder(row pgx.Row) (Order, error) {
	var o Order
	err := row.Scan(&o.ID, &o.PortfolioID, &o.Symbol, &o.Side, &o.Type, &o.Quantity, &o.Status, &o.Price, &o.StopPrice,
		&o.FilledPrice, &o.Fee, &o.FilledAt, &o.CreatedAt, &o.TriggeredBy, &o.RejectReason)
	return o, err
}

func (s *PGStore) Append(userID string, o Order) Order {
	ctx := context.Background()
	var n int64
	if err := s.pool.QueryRow(ctx, `SELECT nextval('order_id_seq')`).Scan(&n); err != nil {
		log.Printf("order: append: %v", err)
		return o
	}
	o.ID = db.HexID("ord_", n)
	_, err := s.pool.Exec(ctx, `INSERT INTO orders (user_id, `+orderCols+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		userID, o.ID, o.PortfolioID, o.Symbol, o.Side, o.Type, o.Quantity, o.Status, o.Price, o.StopPrice,
		o.FilledPrice, o.Fee, o.FilledAt, o.CreatedAt, o.TriggeredBy, o.RejectReason)
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
		if err := rows.Scan(&q.userID, &o.ID, &o.PortfolioID, &o.Symbol, &o.Side, &o.Type, &o.Quantity, &o.Status, &o.Price, &o.StopPrice,
			&o.FilledPrice, &o.Fee, &o.FilledAt, &o.CreatedAt, &o.TriggeredBy, &o.RejectReason); err != nil {
			log.Printf("order: queued: %v", err)
			break
		}
		out = append(out, q)
	}
	return out
}

func (s *PGStore) Replace(userID string, o Order) Order {
	_, err := s.pool.Exec(context.Background(), `UPDATE orders SET status = $3, price = $4, stop_price = $5, filled_price = $6, fee = $7,
		filled_at = $8, created_at = $9, triggered_by = $10, reject_reason = $11 WHERE user_id = $1 AND id = $2`,
		userID, o.ID, o.Status, o.Price, o.StopPrice, o.FilledPrice, o.Fee, o.FilledAt, o.CreatedAt, o.TriggeredBy, o.RejectReason)
	if err != nil {
		log.Printf("order: replace: %v", err)
	}
	return o
}
