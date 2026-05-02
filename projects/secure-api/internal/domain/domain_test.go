package domain_test

import (
	"testing"
	"time"

	"tour_of_go/projects/secure-api/internal/domain"
)

func TestClaims_Immutability(t *testing.T) {
	roles := []string{"admin", "user"}
	exp := time.Now().Add(time.Hour)
	c := domain.NewClaims("u1", roles, exp)

	if c.UserID() != "u1" {
		t.Fatalf("want u1, got %s", c.UserID())
	}
	if c.ExpiresAt() != exp {
		t.Fatal("expiresAt mismatch")
	}

	// Mutating the original slice must not affect Claims.
	roles[0] = "hacked"
	if c.Roles()[0] != "admin" {
		t.Fatal("Claims.roles was mutated via original slice")
	}

	// Mutating the returned slice must not affect Claims.
	got := c.Roles()
	got[0] = "hacked"
	if c.Roles()[0] != "admin" {
		t.Fatal("Claims.roles was mutated via returned slice")
	}
}

func TestClaims_IsExpired(t *testing.T) {
	past := domain.NewClaims("u", nil, time.Now().Add(-time.Second))
	future := domain.NewClaims("u", nil, time.Now().Add(time.Hour))

	if !past.IsExpired() {
		t.Fatal("expected past claims to be expired")
	}
	if future.IsExpired() {
		t.Fatal("expected future claims to not be expired")
	}
}

func TestToken_Immutability(t *testing.T) {
	tok := domain.NewToken("abc.def.ghi", "Bearer", 3600)
	if tok.AccessToken() != "abc.def.ghi" {
		t.Fatalf("want abc.def.ghi, got %s", tok.AccessToken())
	}
	if tok.TokenType() != "Bearer" {
		t.Fatalf("want Bearer, got %s", tok.TokenType())
	}
	if tok.ExpiresIn() != 3600 {
		t.Fatalf("want 3600, got %d", tok.ExpiresIn())
	}
}
