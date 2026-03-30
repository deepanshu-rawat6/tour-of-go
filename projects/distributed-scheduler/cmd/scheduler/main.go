// Command scheduler is the entry point for the distributed job scheduler.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"tour_of_go/projects/distributed-scheduler/internal/adapters/destinations"
	"tour_of_go/projects/distributed-scheduler/internal/adapters/postgres"
	redisadapter "tour_of_go/projects/distributed-scheduler/internal/adapters/redis"
	"tour_of_go/projects/distributed-scheduler/internal/adapters/search"
	"tour_of_go/projects/distributed-scheduler/internal/api"
	"tour_of_go/projects/distributed-scheduler/internal/app"
	"tour_of_go/projects/distributed-scheduler/internal/concurrency"
	"tour_of_go/projects/distributed-scheduler/internal/config"
	"tour_of_go/projects/distributed-scheduler/internal/crons"
	"tour_of_go/projects/distributed-scheduler/internal/domain"
	"tour_of_go/projects/distributed-scheduler/internal/ports"
	"tour_of_go/projects/distributed-scheduler/internal/scheduler"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	// ── Infrastructure ────────────────────────────────────────────────────────
	db, err := sqlx.Connect("postgres", cfg.Database.DSN)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr})

	// ── Adapters ──────────────────────────────────────────────────────────────
	jobRepo := postgres.NewJobRepo(db)
	configRepo := postgres.NewConfigRepo(db)

	bleveSearch, err := search.NewBleveSearch(cfg.Search.IndexPath)
	if err != nil {
		slog.Error("search init failed", "error", err)
		os.Exit(1)
	}

	leaseService := redisadapter.NewLeaseService(redisClient)
	heartbeatService := redisadapter.NewHeartbeatService(redisClient, jobRepo)

	inMemDest := destinations.NewInMemoryDestination(1000)
	destMap := map[domain.DestinationType]ports.Destination{
		domain.DestinationInMemory: inMemDest,
	}

	// ── Core services ─────────────────────────────────────────────────────────
	concurrencyMgr := concurrency.NewManager(func(jobName string) map[string]int {
		c, err := configRepo.GetByJobName(context.Background(), jobName)
		if err != nil {
			return nil
		}
		return c.ConcurrencyRules
	})

	algorithm := scheduler.NewAlgorithm(concurrencyMgr, bleveSearch)
	publisher := scheduler.NewPublisher(concurrencyMgr, jobRepo, destMap)
	schedulerSvc := scheduler.NewService(algorithm, publisher, configRepo, cfg.Scheduler.PoolSize)

	// ── Crons ─────────────────────────────────────────────────────────────────
	coldSched := crons.NewColdScheduler(
		bleveSearch, concurrencyMgr,
		schedulerSvc.TriggerScheduling,
		schedulerSvc.IsPending,
		schedulerSvc.LastAttempt,
		cfg.Scheduler.ColdInterval,
		cfg.Scheduler.StaleAfter,
	)
	concRefresher := crons.NewConcurrencyRefresher(jobRepo, configRepo, concurrencyMgr, cfg.Scheduler.RefreshInterval)
	longPublished := crons.NewLongPublishedChecker(jobRepo, cfg.Scheduler.LongPublishedThreshold, cfg.Scheduler.RefreshInterval)
	hbMarker := crons.NewHeartbeatMarker(heartbeatService, cfg.Scheduler.HeartbeatInterval)
	zombieChecker := crons.NewZombieChecker(heartbeatService, schedulerSvc.TriggerScheduling, cfg.Scheduler.ZombieCheckInterval)

	// ── HTTP API ──────────────────────────────────────────────────────────────
	handler := api.NewHandler(jobRepo, schedulerSvc.TriggerScheduling, concurrencyMgr.Snapshot)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	httpServer := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// ── Application manager ───────────────────────────────────────────────────
	manager := app.NewManager(
		leaseService, concurrencyMgr, bleveSearch,
		jobRepo, configRepo, heartbeatService,
		coldSched, concRefresher, longPublished, hbMarker, zombieChecker,
		httpServer, app.Hostname(),
	)

	if err := manager.Run(); err != nil {
		slog.Error("scheduler exited with error", "error", err)
		os.Exit(1)
	}
}
