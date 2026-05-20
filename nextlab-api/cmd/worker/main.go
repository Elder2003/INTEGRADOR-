package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Elder2003/nextlab-api/internal/config"
	"github.com/Elder2003/nextlab-api/internal/tracing"
)

const (
	streamName    = "nextlab:sessions:jobs"
	consumerGroup = "analysis-workers"
	consumerName  = "worker-1"
	blockDuration = 5 * time.Second
	sweepInterval = 60 * time.Second
	orphanAge     = 60 * time.Second
	claimIdle     = 30 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL
	pgPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer rdb.Close()

	pingCtx, pingCancel := context.WithTimeout(ctx, 10*time.Second)
	defer pingCancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		slog.Error("failed to connect to redis", "addr", cfg.RedisAddr, "err", err)
		os.Exit(1)
	}

	repo := tracing.NewRepository(pgPool, rdb)
	svc := tracing.NewService(repo, rdb)

	// Create consumer group (idempotent — ignore BUSYGROUP error)
	_ = rdb.XGroupCreateMkStream(ctx, streamName, consumerGroup, "0").Err()

	slog.Info("worker started", slog.String("service", "worker"), slog.String("correlation_id", "system"))

	// Re-enqueue orphaned pending jobs from prior crashes
	reEnqueueOrphans(ctx, svc)

	// Background: deadline + timeout sweep
	go runSweep(ctx, svc)

	// Background: re-claim stale in-flight messages
	go runOrphanClaimer(ctx, svc, rdb)

	// Main: consume Redis Streams
	runConsumerLoop(ctx, svc, rdb)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("worker shutting down")
}

// runConsumerLoop reads jobs from the Redis stream and processes them.
func runConsumerLoop(ctx context.Context, svc *tracing.Service, rdb *redis.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		results, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{streamName, ">"},
			Count:    10,
			Block:    blockDuration,
		}).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			slog.ErrorContext(ctx, "xreadgroup error", "err", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range results {
			for _, msg := range stream.Messages {
				processMessage(ctx, svc, rdb, msg)
			}
		}
	}
}

// processMessage handles a single stream message.
func processMessage(ctx context.Context, svc *tracing.Service, rdb *redis.Client, msg redis.XMessage) {
	jobIDStr, _ := msg.Values["job_id"].(string)
	slog.InfoContext(ctx, "processing job", "job_id", jobIDStr, "stream_id", msg.ID)

	if err := processJob(ctx, svc, msg.Values); err != nil {
		slog.ErrorContext(ctx, "job failed", "job_id", jobIDStr, "err", err)
		// Leave in PEL; will be re-claimed by orphan claimer on next sweep
		return
	}

	if err := rdb.XAck(ctx, streamName, consumerGroup, msg.ID).Err(); err != nil {
		slog.ErrorContext(ctx, "xack failed", "job_id", jobIDStr, "err", err)
	}
}

// processJob marks the job processing → done and performs analysis work.
func processJob(ctx context.Context, svc *tracing.Service, values map[string]interface{}) error {
	jobIDStr, _ := values["job_id"].(string)
	sessionIDStr, _ := values["session_id"].(string)

	slog.InfoContext(ctx, "job done",
		slog.String("job_id", jobIDStr),
		slog.String("session_id", sessionIDStr),
	)
	// Placeholder: future analysis work (anti-cheat scoring, risk flags, etc.)
	return nil
}

// reEnqueueOrphans re-enqueues pending jobs that were never XADD'd.
func reEnqueueOrphans(ctx context.Context, svc *tracing.Service) {
	jobs, err := svc.GetOrphanedJobs(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch orphaned jobs", "err", err)
		return
	}
	for _, job := range jobs {
		svc.EnqueueJobForSession(ctx, job.SessionID, job.SubmissionID)
	}
	if len(jobs) > 0 {
		slog.Info("re-enqueued orphaned jobs", "count", len(jobs))
	}
}

// runOrphanClaimer periodically XCLAIM messages idle > claimIdle so another consumer retries them.
func runOrphanClaimer(ctx context.Context, svc *tracing.Service, rdb *redis.Client) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			claimStalePEL(ctx, rdb)
		}
	}
}

func claimStalePEL(ctx context.Context, rdb *redis.Client) {
	entries, _, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   streamName,
		Group:    consumerGroup,
		Consumer: consumerName,
		MinIdle:  claimIdle,
		Start:    "0-0",
		Count:    50,
	}).Result()
	if err != nil {
		slog.ErrorContext(ctx, "xautoclaim failed", "err", err)
		return
	}
	if len(entries) > 0 {
		slog.Info("reclaimed stale PEL entries", "count", len(entries))
	}
}

// runSweep periodically closes sessions whose deadline has passed or that have timed out.
func runSweep(ctx context.Context, svc *tracing.Service) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepDeadlined(ctx, svc)
			sweepTimedOut(ctx, svc)
		}
	}
}

func sweepDeadlined(ctx context.Context, svc *tracing.Service) {
	sessions, err := svc.GetDeadlinedSessions(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch deadlined sessions", "err", err)
		return
	}
	for _, s := range sessions {
		if err := svc.CloseSessionDeadline(ctx, s.ID); err != nil {
			slog.ErrorContext(ctx, "failed to close deadlined session", "session_id", s.ID, "err", err)
		}
	}
}

func sweepTimedOut(ctx context.Context, svc *tracing.Service) {
	sessions, err := svc.GetTimedOutSessions(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch timed-out sessions", "err", err)
		return
	}
	for _, s := range sessions {
		if err := svc.CloseSessionTimeout(ctx, s.ID); err != nil {
			slog.ErrorContext(ctx, "failed to close timed-out session", "session_id", s.ID, "err", err)
		}
	}
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
