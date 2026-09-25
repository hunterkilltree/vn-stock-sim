package watchlist

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore keeps watchlists in Postgres, in the order symbols were added
// (phase-persistence.md). Errors are logged and degrade to "no change" /
// an empty list (decision 9).
type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) Add(userID, symbol string) {
	if _, err := s.pool.Exec(context.Background(),
		`INSERT INTO watchlist_items (user_id, symbol) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, symbol); err != nil {
		log.Printf("watchlist: add: %v", err)
	}
}

func (s *PGStore) Remove(userID, symbol string) {
	if _, err := s.pool.Exec(context.Background(),
		`DELETE FROM watchlist_items WHERE user_id = $1 AND symbol = $2`, userID, symbol); err != nil {
		log.Printf("watchlist: remove: %v", err)
	}
}

func (s *PGStore) List(userID string) []entry {
	rows, err := s.pool.Query(context.Background(),
		`SELECT symbol, added_at FROM watchlist_items WHERE user_id = $1 ORDER BY seq`, userID)
	if err != nil {
		log.Printf("watchlist: list: %v", err)
		return nil
	}
	defer rows.Close()
	var out []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.symbol, &e.addedAt); err != nil {
			log.Printf("watchlist: list: %v", err)
			return out
		}
		out = append(out, e)
	}
	return out
}
