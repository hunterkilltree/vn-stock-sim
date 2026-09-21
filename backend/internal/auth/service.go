package auth

import "github.com/hunterkilltree/vn-stock-sim/backend/internal/authtoken"

// Service holds auth business rules. It depends only on the store and the
// token issuer — no HTTP concerns leak in here (those stay in handler.go).
type Service struct {
	store  *MemoryStore
	tokens *authtoken.Issuer
}

func NewService(store *MemoryStore, tokens *authtoken.Issuer) *Service {
	return &Service{store: store, tokens: tokens}
}

func (s *Service) Register(email, displayName, password string) (authResponse, error) {
	user, err := s.store.Create(email, displayName, password)
	if err != nil {
		return authResponse{}, err
	}
	token, expiresIn := s.tokens.Issue(user.ID, user.Email)
	return authResponse{AccessToken: token, ExpiresIn: expiresIn, User: user}, nil
}

func (s *Service) Login(email, password string) (authResponse, error) {
	user, err := s.store.VerifyCredentials(email, password)
	if err != nil {
		return authResponse{}, err
	}
	token, expiresIn := s.tokens.Issue(user.ID, user.Email)
	return authResponse{AccessToken: token, ExpiresIn: expiresIn, User: user}, nil
}

func (s *Service) Me(userID string) (User, error) {
	return s.store.ByID(userID)
}
