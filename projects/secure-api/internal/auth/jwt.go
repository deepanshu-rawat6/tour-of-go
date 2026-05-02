// Package auth provides JWT-based implementations of the TokenIssuer and TokenValidator ports.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"tour_of_go/projects/secure-api/internal/domain"
)

// ErrTokenExpired is returned when a token's expiry has passed.
var ErrTokenExpired = errors.New("token expired")

// ErrTokenInvalid is returned for malformed or tampered tokens.
var ErrTokenInvalid = errors.New("token invalid")

// JWTAdapter signs and validates HMAC-SHA256 JWTs.
// Implements ports.TokenIssuer and ports.TokenValidator.
type JWTAdapter struct {
	secret []byte
	expiry time.Duration
}

func NewJWTAdapter(secret string, expiry time.Duration) *JWTAdapter {
	return &JWTAdapter{secret: []byte(secret), expiry: expiry}
}

// Issue signs a Claims value and returns a Bearer Token.
func (a *JWTAdapter) Issue(claims domain.Claims) (domain.Token, error) {
	now := time.Now()
	exp := now.Add(a.expiry)
	jc := jwt.MapClaims{
		"sub":   claims.UserID(),
		"roles": claims.Roles(),
		"exp":   exp.Unix(),
		"iat":   now.Unix(),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jc).SignedString(a.secret)
	if err != nil {
		return domain.Token{}, fmt.Errorf("sign token: %w", err)
	}
	return domain.NewToken(raw, "Bearer", int(a.expiry.Seconds())), nil
}

// Validate parses and verifies a raw JWT string, returning immutable Claims.
func (a *JWTAdapter) Validate(raw string) (domain.Claims, error) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return domain.Claims{}, ErrTokenExpired
		}
		return domain.Claims{}, ErrTokenInvalid
	}

	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return domain.Claims{}, ErrTokenInvalid
	}

	sub, _ := mc.GetSubject()
	exp, _ := mc.GetExpirationTime()

	var roles []string
	if rv, ok := mc["roles"].([]any); ok {
		for _, r := range rv {
			if s, ok := r.(string); ok {
				roles = append(roles, s)
			}
		}
	}

	return domain.NewClaims(sub, roles, exp.Time), nil
}
