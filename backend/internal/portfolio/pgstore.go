package portfolio

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db"
)

// PGStore is the Postgres paper-trading ledger (phase-persistence.md):
// portfolios with their cash, positions, and the equity history. Fills
// run in a transaction with the portfolio and position rows locked
// (decision 7), which is what MemoryStore's mutex gave for free.
type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

const portfolioCols = `id, user_id, name, market, kind, starting_capital, currency, created_at`

func scanPortfolio(row pgx.Row) (Portfolio, error) {
	var p Portfolio
	err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Market, &p.Kind, &p.StartingCapital, &p.Currency, &p.CreatedAt)
	return p, err
}

// lockUser serialises portfolio creation per user inside tx, so the
// "first stock trading portfolio becomes the default" rule and the lazy
// default in DefaultFor can't race.
func lockUser(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('portfolio:' || $1))`, userID)
	return err
}

func (s *PGStore) createTx(ctx context.Context, tx pgx.Tx, userID, name, market string, startingCapital float64, currency, kind string) (Portfolio, error) {
	var n int64
	if err := tx.QueryRow(ctx, `SELECT nextval('portfolio_id_seq')`).Scan(&n); err != nil {
		return Portfolio{}, err
	}
	p := Portfolio{
		ID: db.DecID("pf_", n), UserID: userID, Name: name, Market: market, Kind: kind,
		StartingCapital: startingCapital, Currency: currency,
		CreatedAt: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
	// Same default rule as MemoryStore (phase-i.md decision 9): only a
	// stock trading portfolio, and only the user's first one.
	var hasDefault bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM portfolios WHERE user_id = $1 AND is_default)`, userID).Scan(&hasDefault); err != nil {
		return Portfolio{}, err
	}
	isDefault := !hasDefault && kind == KindTrading && market == "stock"
	if _, err := tx.Exec(ctx, `INSERT INTO portfolios (`+portfolioCols+`, cash, is_default) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $6, $9)`,
		p.ID, p.UserID, p.Name, p.Market, p.Kind, p.StartingCapital, p.Currency, p.CreatedAt, isDefault); err != nil {
		return Portfolio{}, err
	}
	// A new portfolio's history starts with its starting capital.
	if _, err := tx.Exec(ctx, `INSERT INTO equity_points (portfolio_id, ts, nav) VALUES ($1, $2, $3)`, p.ID, p.CreatedAt, startingCapital); err != nil {
		return Portfolio{}, err
	}
	return p, nil
}

func (s *PGStore) Create(userID, name, market string, startingCapital float64, currency, kind string) Portfolio {
	var p Portfolio
	err := pgx.BeginFunc(context.Background(), s.pool, func(tx pgx.Tx) error {
		ctx := context.Background()
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		var err error
		p, err = s.createTx(ctx, tx, userID, name, market, startingCapital, currency, kind)
		return err
	})
	if err != nil {
		log.Printf("portfolio: create: %v", err)
	}
	return p
}

// DefaultFor returns the user's default portfolio, opening "Danh mục
// chính" with StartingCash the first time, as MemoryStore does.
func (s *PGStore) DefaultFor(userID string) string {
	var id string
	err := pgx.BeginFunc(context.Background(), s.pool, func(tx pgx.Tx) error {
		ctx := context.Background()
		if err := lockUser(ctx, tx, userID); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `SELECT id FROM portfolios WHERE user_id = $1 AND is_default`, userID).Scan(&id)
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		p, err := s.createTx(ctx, tx, userID, "Danh mục chính", "stock", StartingCash, "VND", KindTrading)
		id = p.ID
		return err
	})
	if err != nil {
		log.Printf("portfolio: default: %v", err)
	}
	return id
}

func (s *PGStore) List(userID string) []Portfolio {
	rows, err := s.pool.Query(context.Background(), `SELECT `+portfolioCols+` FROM portfolios WHERE user_id = $1 ORDER BY seq`, userID)
	if err != nil {
		log.Printf("portfolio: list: %v", err)
		return []Portfolio{}
	}
	defer rows.Close()
	out := []Portfolio{}
	for rows.Next() {
		p, err := scanPortfolio(rows)
		if err != nil {
			log.Printf("portfolio: list: %v", err)
			break
		}
		out = append(out, p)
	}
	return out
}

func (s *PGStore) Get(userID, id string) (Portfolio, bool) {
	p, err := scanPortfolio(s.pool.QueryRow(context.Background(), `SELECT `+portfolioCols+` FROM portfolios WHERE user_id = $1 AND id = $2`, userID, id))
	return p, err == nil
}

func (s *PGStore) OwnerOf(portfolioID string) (string, bool) {
	var userID string
	err := s.pool.QueryRow(context.Background(), `SELECT user_id FROM portfolios WHERE id = $1`, portfolioID).Scan(&userID)
	return userID, err == nil
}

// ApplyFill is MemoryStore.ApplyFill's rules in one transaction: buy
// debits value + fee and blends the average cost, sell credits value −
// fee and shrinks (or closes) the position. Nothing is written when the
// portfolio can't cover the fill.
func (s *PGStore) ApplyFill(portfolioID, sym, side string, quantity, price, fee float64) error {
	return pgx.BeginFunc(context.Background(), s.pool, func(tx pgx.Tx) error {
		ctx := context.Background()
		var cash float64
		err := tx.QueryRow(ctx, `SELECT cash FROM portfolios WHERE id = $1 FOR UPDATE`, portfolioID).Scan(&cash)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var held, avg float64
		err = tx.QueryRow(ctx, `SELECT quantity, avg_cost FROM positions WHERE portfolio_id = $1 AND symbol = $2 FOR UPDATE`, portfolioID, sym).Scan(&held, &avg)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		value := price * quantity

		switch side {
		case "buy":
			if cash < value+fee {
				return ErrInsufficientCash
			}
			newQty := held + quantity
			newAvg := (avg*held + value) / newQty
			if _, err := tx.Exec(ctx, `INSERT INTO positions (portfolio_id, symbol, quantity, avg_cost) VALUES ($1, $2, $3, $4)
				ON CONFLICT (portfolio_id, symbol) DO UPDATE SET quantity = EXCLUDED.quantity, avg_cost = EXCLUDED.avg_cost`,
				portfolioID, sym, newQty, newAvg); err != nil {
				return err
			}
			cash -= value + fee
		case "sell":
			if held < quantity-qtyEpsilon {
				return ErrInsufficientShares
			}
			left := held - quantity
			if left <= qtyEpsilon {
				_, err = tx.Exec(ctx, `DELETE FROM positions WHERE portfolio_id = $1 AND symbol = $2`, portfolioID, sym)
			} else {
				_, err = tx.Exec(ctx, `UPDATE positions SET quantity = $3 WHERE portfolio_id = $1 AND symbol = $2`, portfolioID, sym, left)
			}
			if err != nil {
				return err
			}
			cash += value - fee
		}
		_, err = tx.Exec(ctx, `UPDATE portfolios SET cash = $2 WHERE id = $1`, portfolioID, cash)
		return err
	})
}

func (s *PGStore) Cash(portfolioID string) float64 {
	var cash float64
	if err := s.pool.QueryRow(context.Background(), `SELECT cash FROM portfolios WHERE id = $1`, portfolioID).Scan(&cash); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("portfolio: cash: %v", err)
	}
	return cash
}

func (s *PGStore) Positions(portfolioID string) map[string]position {
	out := map[string]position{}
	rows, err := s.pool.Query(context.Background(), `SELECT symbol, quantity, avg_cost FROM positions WHERE portfolio_id = $1`, portfolioID)
	if err != nil {
		log.Printf("portfolio: positions: %v", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var sym string
		var p position
		if err := rows.Scan(&sym, &p.quantity, &p.avgCost); err != nil {
			log.Printf("portfolio: positions: %v", err)
			break
		}
		out[sym] = p
	}
	return out
}

func (s *PGStore) AppendEquitySnapshot(portfolioID string, nav float64) {
	if _, err := s.pool.Exec(context.Background(), `INSERT INTO equity_points (portfolio_id, ts, nav) VALUES ($1, $2, $3)`,
		portfolioID, time.Now().UTC().Format("2006-01-02T15:04:05Z"), nav); err != nil {
		log.Printf("portfolio: equity: %v", err)
	}
}

func (s *PGStore) EquityHistory(portfolioID string) []EquityPoint {
	out := []EquityPoint{}
	rows, err := s.pool.Query(context.Background(), `SELECT ts, nav FROM equity_points WHERE portfolio_id = $1 ORDER BY id`, portfolioID)
	if err != nil {
		log.Printf("portfolio: equity history: %v", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var e EquityPoint
		if err := rows.Scan(&e.Timestamp, &e.NAV); err != nil {
			log.Printf("portfolio: equity history: %v", err)
			break
		}
		out = append(out, e)
	}
	return out
}
