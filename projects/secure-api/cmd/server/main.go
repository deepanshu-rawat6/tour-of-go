// Command server starts the secure-api HTTP server.
// All dependencies are wired here via constructor injection (Dependency Inversion Principle).
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"tour_of_go/projects/secure-api/internal/auth"
	"tour_of_go/projects/secure-api/internal/config"
	"tour_of_go/projects/secure-api/internal/handler"
	"tour_of_go/projects/secure-api/internal/middleware"
)

func main() {
	addr := envOr("ADDR", ":8080")
	secret := envOr("JWT_SECRET", "change-me")
	expiry := 1 * time.Hour

	// Build adapters (concrete implementations of port interfaces)
	jwtAdapter := auth.NewJWTAdapter(secret, expiry)
	userStore := auth.NewUserStore(expiry)
	_ = userStore.AddUser("admin", "secret", []string{"admin", "user"})
	_ = userStore.AddUser("alice", "password", []string{"user"})

	// Build router — Open/Closed: add routes without touching existing handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.Health)
	mux.HandleFunc("/oauth2/token", handler.Token(userStore, jwtAdapter))
	mux.Handle("/me", middleware.Chain(
		http.HandlerFunc(handler.Me),
		middleware.Auth(jwtAdapter),
	))

	// Optionally start with mTLS
	tlsCfg, err := config.LoadTLS(
		envOr("TLS_CERT", ""),
		envOr("TLS_KEY", ""),
		envOr("TLS_CA", ""),
	)
	if err != nil {
		log.Fatalf("load TLS: %v", err)
	}

	srv := &http.Server{Addr: addr, Handler: mux, TLSConfig: tlsCfg}

	if tlsCfg != nil {
		log.Printf("secure-api listening on %s (mTLS)", addr)
		log.Fatal(srv.ListenAndServeTLS(
			envOr("TLS_CERT", "certs/server.crt"),
			envOr("TLS_KEY", "certs/server.key"),
		))
	} else {
		log.Printf("secure-api listening on %s", addr)
		log.Fatal(srv.ListenAndServe())
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
