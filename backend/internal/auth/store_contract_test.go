package auth

import (
	"errors"
	"os"
	"testing"

	"github.com/hunterkilltree/vn-stock-sim/backend/internal/db/dbtest"
)

// Both stores must behave the same (phase-persistence.md): the Postgres
// run needs TEST_DATABASE_URL.
func eachStore(t *testing.T, run func(t *testing.T, s Store)) {
	t.Run("memory", func(t *testing.T) { run(t, NewMemoryStore()) })
	if os.Getenv("TEST_DATABASE_URL") != "" {
		t.Run("postgres", func(t *testing.T) { run(t, NewPGStore(dbtest.Pool(t, "auth"))) })
	}
}

func TestStoreContract(t *testing.T) {
	eachStore(t, func(t *testing.T, s Store) {
		u, err := s.Create("an@test.vn", "An", "Password123", "both")
		if err != nil || u.ID == "" || u.ID[0] != 'u' {
			t.Fatalf("create: %+v %v", u, err)
		}
		if _, err := s.Create("an@test.vn", "An 2", "x", ""); !errors.Is(err, ErrEmailTaken) {
			t.Fatalf("duplicate email: want ErrEmailTaken, got %v", err)
		}
		if got, err := s.VerifyCredentials("an@test.vn", "Password123"); err != nil || got != u {
			t.Fatalf("verify: %+v %v", got, err)
		}
		if _, err := s.VerifyCredentials("an@test.vn", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("wrong password: %v", err)
		}
		if _, err := s.VerifyCredentials("nobody@test.vn", "x"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("unknown email: %v", err)
		}
		if got, err := s.ByID(u.ID); err != nil || got.MarketInterest != "both" {
			t.Fatalf("by id: %+v %v", got, err)
		}
		if _, err := s.ByID("u999999"); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("missing id: %v", err)
		}
	})
}
