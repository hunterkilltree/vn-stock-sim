package replay

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db"
)

// PGStore keeps each Replay session as one JSONB document -- the session
// is always read and written whole, and its bars are the history it was
// started with (phase-persistence.md decisions 6 and 8).
type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) nextSessionID() string {
	var n int64
	if err := s.pool.QueryRow(context.Background(), `SELECT nextval('replay_id_seq')`).Scan(&n); err != nil {
		log.Printf("replay: next id: %v", err)
	}
	return db.DecID("rp_", n)
}

func (s *PGStore) Append(userID string, sess *Session) {
	body, err := json.Marshal(sess)
	if err == nil {
		_, err = s.pool.Exec(context.Background(), `INSERT INTO replay_sessions (id, user_id, body) VALUES ($1, $2, $3)`, sess.ID, userID, body)
	}
	if err != nil {
		log.Printf("replay: append: %v", err)
	}
}

func (s *PGStore) Get(userID, id string) (*Session, bool) {
	var body []byte
	if err := s.pool.QueryRow(context.Background(), `SELECT body FROM replay_sessions WHERE user_id = $1 AND id = $2`, userID, id).Scan(&body); err != nil {
		return nil, false
	}
	var sess Session
	if err := json.Unmarshal(body, &sess); err != nil {
		log.Printf("replay: get: %v", err)
		return nil, false
	}
	return &sess, true
}

func (s *PGStore) Save(sess *Session) {
	body, err := json.Marshal(sess)
	if err == nil {
		_, err = s.pool.Exec(context.Background(), `UPDATE replay_sessions SET body = $3, updated_at = now() WHERE user_id = $1 AND id = $2`, sess.UserID, sess.ID, body)
	}
	if err != nil {
		log.Printf("replay: save: %v", err)
	}
}
