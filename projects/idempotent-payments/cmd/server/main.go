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

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tour_of_go/projects/idempotent-payments/internal/adapters/postgres"
	"tour_of_go/projects/idempotent-payments/internal/config"
	"tour_of_go/projects/idempotent-payments/internal/handler"
	"tour_of_go/projects/idempotent-payments/internal/middleware"
	"tour_of_go/projects/idempotent-payments/internal/service"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	ledgerRepo := postgres.NewLedgerRepo(pool)
	idempotencyRepo := postgres.NewIdempotencyRepo(pool)
	svc := service.NewLedgerService(ledgerRepo)

	// TTL cleanup goroutine.
	go func() {
		ticker := time.NewTicker(cfg.Idempotency.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n, err := idempotencyRepo.DeleteExpired(context.Background())
				if err != nil {
					log.Printf("cleanup error: %v", err)
				} else if n > 0 {
					log.Printf("deleted %d expired idempotency keys", n)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	r := chi.NewRouter()
	r.With(middleware.Idempotency(idempotencyRepo, cfg.Idempotency.TTL)).
		Post("/payments", handler.Payment(svc))
	r.Get("/accounts/{id}", handler.GetAccount(ledgerRepo))

	srv := &http.Server{Addr: cfg.Server.Addr, Handler: r}

	go func() {
		fmt.Printf("listening on %s\n", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx) //nolint:errcheck
	fmt.Println("server stopped")
}
