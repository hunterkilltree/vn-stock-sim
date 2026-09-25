package backtest

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db"
)

// PGStore keeps each backtest run as one JSONB body, equity curve and
// trades included (phase-persistence.md decision 6).
type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) Append(userID string, b Backtest) Backtest {
	ctx := context.Background()
	var n int64
	if err := s.pool.QueryRow(ctx, `SELECT nextval('backtest_id_seq')`).Scan(&n); err != nil {
		log.Printf("backtest: append: %v", err)
		return b
	}
	b.ID = db.HexID("bt_", n)
	body, err := json.Marshal(b)
	if err == nil {
		_, err = s.pool.Exec(ctx, `INSERT INTO backtests (id, user_id, body) VALUES ($1, $2, $3)`, b.ID, userID, body)
	}
	if err != nil {
		log.Printf("backtest: append: %v", err)
	}
	return b
}

func (s *PGStore) List(userID string) []Backtest {
	rows, err := s.pool.Query(context.Background(), `SELECT body FROM backtests WHERE user_id = $1 ORDER BY seq`, userID)
	if err != nil {
		log.Printf("backtest: list: %v", err)
		return []Backtest{}
	}
	defer rows.Close()
	out := []Backtest{}
	for rows.Next() {
		var body []byte
		var b Backtest
		if rows.Scan(&body) == nil && json.Unmarshal(body, &b) == nil {
			out = append(out, b)
		}
	}
	return out
}

func (s *PGStore) ByID(userID, id string) (Backtest, bool) {
	var body []byte
	if err := s.pool.QueryRow(context.Background(), `SELECT body FROM backtests WHERE user_id = $1 AND id = $2`, userID, id).Scan(&body); err != nil {
		return Backtest{}, false
	}
	var b Backtest
	return b, json.Unmarshal(body, &b) == nil
}
