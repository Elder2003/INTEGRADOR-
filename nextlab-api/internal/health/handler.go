package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

// EmbedChecker is the minimal interface needed from the embed client.
type EmbedChecker interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type DependencyStatus struct {
	Status    string `json:"status"`
	LatencyMs int    `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
	Model     string `json:"model,omitempty"`
	Dim       int    `json:"dim,omitempty"`
}

type HealthReport struct {
	Status       string                      `json:"status"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
}

// MLMetaProvider surfaces the active ML model metadata for health reporting.
type MLMetaProvider interface {
	GetActiveMeta() MLModelMeta
}

// MLModelMeta is the minimal ML model info needed for the health check.
type MLModelMeta struct {
	Available bool
	Version   int
	TrainedAt string
}

type Checker struct {
	db          *pgxpool.Pool
	redis       *redis.Client
	minio       *minio.Client
	ollamaURL   string
	ollamaModel string
	embedClient EmbedChecker
	mlMeta      MLMetaProvider
}

func NewChecker(db *pgxpool.Pool, redis *redis.Client, minio *minio.Client, ollamaURL, ollamaModel string, embedClient EmbedChecker) *Checker {
	return &Checker{
		db:          db,
		redis:       redis,
		minio:       minio,
		ollamaURL:   ollamaURL,
		ollamaModel: ollamaModel,
		embedClient: embedClient,
	}
}

// SetMLMetaProvider wires the ML service for health reporting.
func (c *Checker) SetMLMetaProvider(p MLMetaProvider) {
	c.mlMeta = p
}

func (c *Checker) Check(ctx context.Context) *HealthReport {
	infraCtx, infraCancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer infraCancel()

	aiCtx, aiCancel := context.WithTimeout(ctx, 2*time.Second)
	defer aiCancel()

	report := &HealthReport{
		Dependencies: make(map[string]DependencyStatus),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	numChecks := 5
	if c.mlMeta != nil {
		numChecks = 6
	}
	wg.Add(numChecks)

	set := func(key string, status DependencyStatus) {
		mu.Lock()
		report.Dependencies[key] = status
		mu.Unlock()
	}

	go func() {
		defer wg.Done()
		start := time.Now()
		if err := c.db.Ping(infraCtx); err != nil {
			set("database", DependencyStatus{Status: "down", Error: err.Error()})
		} else {
			set("database", DependencyStatus{Status: "ok", LatencyMs: int(time.Since(start).Milliseconds())})
		}
	}()

	go func() {
		defer wg.Done()
		start := time.Now()
		if err := c.redis.Ping(infraCtx).Err(); err != nil {
			set("cache", DependencyStatus{Status: "down", Error: err.Error()})
		} else {
			set("cache", DependencyStatus{Status: "ok", LatencyMs: int(time.Since(start).Milliseconds())})
		}
	}()

	go func() {
		defer wg.Done()
		start := time.Now()
		if _, err := c.minio.BucketExists(infraCtx, "test"); err != nil {
			set("storage", DependencyStatus{Status: "down", Error: err.Error()})
		} else {
			set("storage", DependencyStatus{Status: "ok", LatencyMs: int(time.Since(start).Milliseconds())})
		}
	}()

	go func() {
		defer wg.Done()
		if c.ollamaURL == "" {
			set("llm_local", DependencyStatus{Status: "down", Error: "ollama URL not configured"})
			return
		}
		payload, _ := json.Marshal(map[string]any{
			"model": c.ollamaModel, "prompt": "ping", "stream": false,
		})
		start := time.Now()
		req, err := http.NewRequestWithContext(aiCtx, "POST", c.ollamaURL+"/api/generate", bytes.NewReader(payload))
		if err != nil {
			set("llm_local", DependencyStatus{Status: "down", Error: err.Error()})
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			set("llm_local", DependencyStatus{Status: "down", Error: err.Error()})
			return
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)
		if resp.StatusCode != http.StatusOK {
			set("llm_local", DependencyStatus{Status: "down", Error: fmt.Sprintf("HTTP %d", resp.StatusCode)})
			return
		}
		set("llm_local", DependencyStatus{
			Status:    "ok",
			LatencyMs: int(time.Since(start).Milliseconds()),
			Model:     c.ollamaModel,
		})
	}()

	go func() {
		defer wg.Done()
		if c.embedClient == nil {
			set("embed_model", DependencyStatus{Status: "down", Error: "embed client not configured"})
			return
		}
		start := time.Now()
		vec, err := c.embedClient.Embed(aiCtx, "health-probe")
		if err != nil {
			set("embed_model", DependencyStatus{Status: "down", Error: err.Error()})
			return
		}
		set("embed_model", DependencyStatus{
			Status:    "ok",
			LatencyMs: int(time.Since(start).Milliseconds()),
			Dim:       len(vec),
		})
	}()

	// ML model check (non-critical: failure → aiDown, not criticalDown)
	if c.mlMeta != nil {
		go func() {
			defer wg.Done()
			meta := c.mlMeta.GetActiveMeta()
			if !meta.Available {
				set("ml_model", DependencyStatus{Status: "down", Error: "no model loaded"})
			} else {
				set("ml_model", DependencyStatus{Status: "ok", Model: fmt.Sprintf("v%d trained %s", meta.Version, meta.TrainedAt)})
			}
		}()
	}

	wg.Wait()

	// Determine overall status
	criticalDown := false
	aiDown := false
	for key, dep := range report.Dependencies {
		if dep.Status != "ok" {
			switch key {
			case "database", "cache", "storage":
				criticalDown = true
			case "llm_local", "embed_model", "ml_model":
				aiDown = true
			}
		}
	}

	switch {
	case criticalDown:
		report.Status = "down"
	case aiDown:
		report.Status = "degraded"
	default:
		report.Status = "ok"
	}

	return report
}

func (r *HealthReport) StatusCode() int {
	if r.Status == "down" {
		return 503
	}
	return 200
}

func (r *HealthReport) JSON() []byte {
	data, _ := json.Marshal(r)
	return data
}
