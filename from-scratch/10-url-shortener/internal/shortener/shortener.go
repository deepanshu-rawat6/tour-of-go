// Package shortener generates and resolves short URL codes.
package shortener

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

var ErrNotFound = errors.New("short code not found")

// Code generates a random 6-character URL-safe code.
func Code() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base64.URLEncoding.EncodeToString(b)[:6]
	return strings.ReplaceAll(code, "=", "x"), nil
}
