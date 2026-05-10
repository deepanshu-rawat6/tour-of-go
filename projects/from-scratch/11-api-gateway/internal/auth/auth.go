package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid = errors.New("token invalid")
	ErrTokenExpired = errors.New("token expired")
)

// Claims holds the validated identity extracted from a JWT.
type Claims struct {
	UserID string
	Role   string
}

// Authenticator holds the HMAC secret in memory and issues/validates HS256 JWTs.
type Authenticator struct {
	secret []byte
	expiry time.Duration
}

func NewAuthenticator(secret string, expiry time.Duration) *Authenticator {
	return &Authenticator{secret: []byte(secret), expiry: expiry}
}

func (a *Authenticator) Issue(userID, role string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  now.Add(a.expiry).Unix(),
		"iat":  now.Unix(),
	})
	raw, err := token.SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return raw, nil
}

func (a *Authenticator) Validate(raw string) (Claims, error) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return Claims{}, ErrTokenExpired
		}
		return Claims{}, ErrTokenInvalid
	}

	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return Claims{}, ErrTokenInvalid
	}
	userID, _ := mc["sub"].(string)
	role, _ := mc["role"].(string)
	return Claims{UserID: userID, Role: role}, nil
}
