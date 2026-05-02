package auth_test

import (
	"strings"
	"testing"
	"time"

	"tour_of_go/projects/secure-api/internal/auth"
	"tour_of_go/projects/secure-api/internal/domain"
)

func TestJWTAdapter_IssueAndValidate(t *testing.T) {
	adapter := auth.NewJWTAdapter("test-secret", time.Hour)
	claims := domain.NewClaims("user42", []string{"admin"}, time.Now().Add(time.Hour))

	tok, err := adapter.Issue(claims)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if tok.AccessToken() == "" {
		t.Fatal("expected non-empty access token")
	}
	if tok.TokenType() != "Bearer" {
		t.Fatalf("want Bearer, got %s", tok.TokenType())
	}

	got, err := adapter.Validate(tok.AccessToken())
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got.UserID() != "user42" {
		t.Fatalf("want user42, got %s", got.UserID())
	}
	if len(got.Roles()) != 1 || got.Roles()[0] != "admin" {
		t.Fatalf("unexpected roles: %v", got.Roles())
	}
}

func TestJWTAdapter_Validate_TableDriven(t *testing.T) {
	adapter := auth.NewJWTAdapter("secret", time.Hour)
	claims := domain.NewClaims("u1", []string{"user"}, time.Now().Add(time.Hour))
	validTok, _ := adapter.Issue(claims)

	// Issue an expired token using a -1s expiry adapter
	expiredAdapter := auth.NewJWTAdapter("secret", -time.Second)
	expiredClaims := domain.NewClaims("u1", nil, time.Now().Add(-time.Second))
	expiredTok, _ := expiredAdapter.Issue(expiredClaims)

	tests := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{"valid token", validTok.AccessToken(), nil},
		{"expired token", expiredTok.AccessToken(), auth.ErrTokenExpired},
		{"malformed token", "not.a.jwt", auth.ErrTokenInvalid},
		{"wrong signature", validTok.AccessToken()[:len(validTok.AccessToken())-4] + "XXXX", auth.ErrTokenInvalid},
		{"empty string", "", auth.ErrTokenInvalid},
		{"wrong secret", func() string {
			a2 := auth.NewJWTAdapter("other-secret", time.Hour)
			t2, _ := a2.Issue(claims)
			return t2.AccessToken()
		}(), auth.ErrTokenInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := adapter.Validate(tc.raw)
			if tc.wantErr == nil && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
			if tc.wantErr != nil && err != tc.wantErr {
				// expired tokens may also surface as invalid depending on truncation
				if tc.wantErr == auth.ErrTokenExpired && strings.Contains(err.Error(), "expired") {
					return
				}
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}
