package health

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

type DependencyStatus struct {
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

type HealthReport struct {
	Status       string                    `json:"status"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
}

type Checker struct {
	db    *pgx.Conn
	redis *redis.Client
	minio *minio.Client
}

func NewChecker(db *pgx.Conn, redis *redis.Client, minio *minio.Client) *Checker {
	return &Checker{
		db:    db,
		redis: redis,
		minio: minio,
	}
}

func (c *Checker) Check(ctx context.Context) *HealthReport {
	// Create a shared deadline context of 400ms
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()

	report := &HealthReport{
		Dependencies: make(map[string]DependencyStatus),
	}

	// Run all pings in parallel
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		start := time.Now()
		err := c.db.Ping(ctxWithTimeout)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			report.Dependencies["database"] = DependencyStatus{
				Status: "down",
				Error:  err.Error(),
			}
		} else {
			report.Dependencies["database"] = DependencyStatus{
				Status:    "ok",
				LatencyMs: int(latency),
			}
		}
	}()

	go func() {
		defer wg.Done()
		start := time.Now()
		err := c.redis.Ping(ctxWithTimeout).Err()
		latency := time.Since(start).Milliseconds()

		if err != nil {
			report.Dependencies["cache"] = DependencyStatus{
				Status: "down",
				Error:  err.Error(),
			}
		} else {
			report.Dependencies["cache"] = DependencyStatus{
				Status:    "ok",
				LatencyMs: int(latency),
			}
		}
	}()

	go func() {
		defer wg.Done()
		start := time.Now()
		_, err := c.minio.BucketExists(ctxWithTimeout, "test")
		latency := time.Since(start).Milliseconds()

		// BucketExists returns no error if successful or if bucket doesn't exist
		// We're just checking connectivity
		if err != nil {
			report.Dependencies["storage"] = DependencyStatus{
				Status: "down",
				Error:  err.Error(),
			}
		} else {
			report.Dependencies["storage"] = DependencyStatus{
				Status:    "ok",
				LatencyMs: int(latency),
			}
		}
	}()

	wg.Wait()

	// Determine aggregate status
	report.Status = "ok"
	for _, dep := range report.Dependencies {
		if dep.Status != "ok" {
			report.Status = "degraded"
			break
		}
	}

	return report
}

func (r *HealthReport) StatusCode() int {
	if r.Status == "ok" {
		return 200
	}
	return 503
}

func (r *HealthReport) JSON() []byte {
	data, _ := json.Marshal(r)
	return data
}
