package auth_test

import (
	"testing"
	"time"

	"tour_of_go/projects/secure-api/internal/auth"
)

func TestUserStore_Authenticate(t *testing.T) {
	store := auth.NewUserStore(time.Hour)
	if err := store.AddUser("admin", "secret", []string{"admin"}); err != nil {
		t.Fatalf("AddUser: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{"valid credentials", "admin", "secret", nil},
		{"wrong password", "admin", "wrong", auth.ErrBadCredentials},
		{"unknown user", "nobody", "secret", auth.ErrBadCredentials},
		{"empty password", "admin", "", auth.ErrBadCredentials},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := store.Authenticate(tc.username, tc.password)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("want %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if claims.UserID() != tc.username {
				t.Fatalf("want %s, got %s", tc.username, claims.UserID())
			}
			if claims.IsExpired() {
				t.Fatal("claims should not be expired")
			}
		})
	}
}
