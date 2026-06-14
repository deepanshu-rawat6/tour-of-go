package auth_test

import (
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/11-api-gateway/internal/auth"
)

func newAuth() *auth.Authenticator {
	return auth.NewAuthenticator("test-secret", time.Hour)
}

func TestIssueAndValidate(t *testing.T) {
	a := newAuth()
	token, err := a.Issue("alice", "admin")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := a.Validate(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "alice" {
		t.Fatalf("want alice, got %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Fatalf("want admin, got %s", claims.Role)
	}
}

func TestValidate_TamperedToken(t *testing.T) {
	a := newAuth()
	token, _ := a.Issue("alice", "user")
	_, err := a.Validate(token + "tampered")
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestValidate_WrongSecret(t *testing.T) {
	issuer := auth.NewAuthenticator("secret-a", time.Hour)
	validator := auth.NewAuthenticator("secret-b", time.Hour)
	token, _ := issuer.Issue("alice", "user")
	_, err := validator.Validate(token)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestValidate_ExpiredToken(t *testing.T) {
	a := auth.NewAuthenticator("test-secret", -time.Second) // already expired
	token, _ := a.Issue("alice", "user")
	_, err := a.Validate(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}
