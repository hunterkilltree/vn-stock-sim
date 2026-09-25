package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db"
)

// PGStore keeps users in Postgres (phase-persistence.md). Same rules as
// MemoryStore: unique email, bcrypt hash, "u<n>" IDs.
type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) Create(email, displayName, password, marketInterest string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	ctx := context.Background()
	var n int64
	if err := s.pool.QueryRow(ctx, `SELECT nextval('user_id_seq')`).Scan(&n); err != nil {
		return User{}, err
	}
	u := User{ID: db.DecID("u", n), Email: email, DisplayName: displayName, MarketInterest: marketInterest}
	_, err = s.pool.Exec(ctx, `INSERT INTO users (id, email, display_name, market_interest, password_hash) VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Email, u.DisplayName, u.MarketInterest, hash)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation on email
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *PGStore) VerifyCredentials(email, password string) (User, error) {
	var u User
	var hash []byte
	err := s.pool.QueryRow(context.Background(),
		`SELECT id, email, display_name, market_interest, password_hash FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &u.DisplayName, &u.MarketInterest, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
		return User{}, ErrInvalidCredentials
	}
	return u, nil
}

func (s *PGStore) ByID(id string) (User, error) {
	var u User
	err := s.pool.QueryRow(context.Background(),
		`SELECT id, email, display_name, market_interest FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.DisplayName, &u.MarketInterest)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return u, err
}
