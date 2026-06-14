package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/chi/v5"
	ratelimit "tour_of_go/projects/from-scratch/04-rate-limiter/ratelimit"

	"tour_of_go/projects/from-scratch/11-api-gateway/internal/auth"
	"tour_of_go/projects/from-scratch/11-api-gateway/internal/config"
	"tour_of_go/projects/from-scratch/11-api-gateway/internal/handler"
	"tour_of_go/projects/from-scratch/11-api-gateway/internal/middleware"
	"tour_of_go/projects/from-scratch/11-api-gateway/internal/proxy"
)

func main() {
	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	authn := auth.NewAuthenticator(cfg.JWT.Secret, cfg.JWT.Expiry)
	limiter := ratelimit.NewTokenBucket(cfg.RateLimit.Rate, cfg.RateLimit.Burst)

	r := chi.NewRouter()

	// Global middleware applied to every request.
	r.Use(middleware.RequestID)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	// Public routes — no auth required.
	r.Group(func(r chi.Router) {
		r.Get("/health", handler.HealthCheck)
		r.Post("/login", handler.Login(authn))
	})

	// Protected API routes — JWT + rate limiting.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(authn))
		r.Use(middleware.RateLimit(limiter))

		for service, upstream := range cfg.Upstreams {
			r.Mount("/"+service, proxy.New(upstream))
		}
	})

	// Admin routes — JWT + admin role required.
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.Auth(authn))
		r.Use(middleware.RequireRole("admin"))
		r.Get("/routes", handler.ListRoutes(cfg.Upstreams))
	})

	srv := &http.Server{Addr: cfg.Server.Addr, Handler: r}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		fmt.Printf("gateway listening on %s\n", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx) //nolint:errcheck
	fmt.Println("gateway stopped")
}
