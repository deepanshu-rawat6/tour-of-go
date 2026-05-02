package domain

import "time"

// Claims is an immutable value object representing an authenticated identity.
// All fields are set at construction; there are no setters.
type Claims struct {
	userID    string
	roles     []string
	expiresAt time.Time
}

func NewClaims(userID string, roles []string, expiresAt time.Time) Claims {
	r := make([]string, len(roles))
	copy(r, roles)
	return Claims{userID: userID, roles: r, expiresAt: expiresAt}
}

func (c Claims) UserID() string      { return c.userID }
func (c Claims) ExpiresAt() time.Time { return c.expiresAt }

// Roles returns a copy so callers cannot mutate the internal slice.
func (c Claims) Roles() []string {
	r := make([]string, len(c.roles))
	copy(r, c.roles)
	return r
}

func (c Claims) IsExpired() bool { return time.Now().After(c.expiresAt) }
