// Package ports defines the application's port interfaces following SOLID principles.
//
// Interface Segregation Principle: each interface has exactly one method.
// Dependency Inversion Principle: high-level packages depend on these abstractions,
// not on concrete implementations.
package ports

import "tour_of_go/projects/secure-api/internal/domain"

// TokenIssuer signs a Claims value and returns a Token. (SRP: only issues tokens)
type TokenIssuer interface {
	Issue(claims domain.Claims) (domain.Token, error)
}

// TokenValidator parses and validates a raw token string. (SRP: only validates)
type TokenValidator interface {
	Validate(raw string) (domain.Claims, error)
}

// UserAuthenticator verifies credentials and returns the user's Claims. (SRP: only authenticates)
type UserAuthenticator interface {
	Authenticate(username, password string) (domain.Claims, error)
}
