package auth

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"tour_of_go/projects/secure-api/internal/domain"
)

// ErrBadCredentials is returned when username/password don't match.
var ErrBadCredentials = errors.New("invalid username or password")

// user is an internal record stored in the UserStore.
type user struct {
	hashedPassword string
	roles          []string
}

// UserStore implements ports.UserAuthenticator using an in-memory user map.
// Dependency Inversion: callers depend on the UserAuthenticator interface, not this struct.
type UserStore struct {
	users  map[string]user
	expiry time.Duration
}

func NewUserStore(expiry time.Duration) *UserStore {
	return &UserStore{users: make(map[string]user), expiry: expiry}
}

// AddUser hashes the password and stores the user. Used at startup to seed users.
func (s *UserStore) AddUser(username, plainPassword string, roles []string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.MinCost)
	if err != nil {
		return err
	}
	s.users[username] = user{hashedPassword: string(hash), roles: roles}
	return nil
}

// Authenticate verifies credentials and returns immutable Claims on success.
func (s *UserStore) Authenticate(username, password string) (domain.Claims, error) {
	u, ok := s.users[username]
	if !ok {
		return domain.Claims{}, ErrBadCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.hashedPassword), []byte(password)); err != nil {
		return domain.Claims{}, ErrBadCredentials
	}
	return domain.NewClaims(username, u.roles, time.Now().Add(s.expiry)), nil
}
