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

	pgadapter "tour_of_go/projects/triage-engine/internal/adapters/postgres"
	"tour_of_go/projects/triage-engine/internal/adapters/diagnostic"
	"tour_of_go/projects/triage-engine/internal/adapters/notifier"
	oaiclient "tour_of_go/projects/triage-engine/internal/adapters/openai"
	"tour_of_go/projects/triage-engine/internal/config"
	"tour_of_go/projects/triage-engine/internal/graph"
	"tour_of_go/projects/triage-engine/internal/handler"
)

func main() {
	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
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

	llm := oaiclient.NewClient(cfg.OpenAI.APIKey, cfg.OpenAI.Model, cfg.OpenAI.EmbeddingModel)
	checkpointer := pgadapter.NewCheckpointer(pool)
	kb := pgadapter.NewKnowledgeRepo(pool, llm)
	diag := diagnostic.NewClient(cfg.Diagnostic.BaseURL)
	n := notifier.NewLogNotifier(nil)

	engine := graph.NewTriageEngine(checkpointer, kb, diag, llm, n)

	r := chi.NewRouter()
	r.Post("/webhooks/ticket", handler.WebhookTicket(engine))
	r.Post("/graph/resume", handler.ResumeGraph(engine))

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
