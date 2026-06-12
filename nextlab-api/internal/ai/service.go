package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/Elder2003/nextlab-api/internal/config"
)

type cachedMode struct {
	mode      string
	fetchedAt time.Time
}

// Service orchestrates mode caching, dispatch, and mode management.
type Service struct {
	repo          *Repository
	primaryClient *BackendClient
	cloudClient   *BackendClient
	defaultModel  string
	modeCache     atomic.Pointer[cachedMode]
}

func NewService(repo *Repository, cfg *config.Config) *Service {
	timeout := time.Duration(cfg.AIPrimaryTimeoutSecs) * time.Second

	svc := &Service{
		repo:         repo,
		defaultModel: cfg.AIPrimaryModel,
		primaryClient: &BackendClient{
			BaseURL:    cfg.AIPrimaryURL,
			HTTPClient: &http.Client{Timeout: timeout * 2}, // outer; per-request ctx controls actual timeout
		},
		cloudClient: &BackendClient{
			BaseURL:    cfg.AICloudURL,
			APIKey:     cfg.AICloudAPIKey,
			HTTPClient: &http.Client{Timeout: 60 * time.Second},
		},
	}
	svc.modeCache.Store(&cachedMode{mode: "local", fetchedAt: time.Now()})
	return svc
}

// SeedIfEmpty inserts the default 'local' mode row if router_configs is empty.
func (s *Service) SeedIfEmpty(ctx context.Context) {
	mode, err := s.repo.GetCurrentMode(ctx)
	if err != nil {
		slog.Warn("ai router: could not read mode for seed check", "err", err)
		return
	}
	// GetCurrentMode returns "local" on ErrNoRows — check if a real row exists
	cfg, err := s.repo.GetCurrentConfig(ctx)
	if err != nil {
		slog.Warn("ai router: could not read config for seed check", "err", err)
		return
	}
	if cfg == nil {
		if _, err := s.repo.InsertModeChange(ctx, "local", nil, nil); err != nil {
			slog.Warn("ai router: failed to seed initial mode", "err", err)
			return
		}
		slog.Info("ai router: seeded initial mode", "mode", "local")
	}
	s.modeCache.Store(&cachedMode{mode: mode, fetchedAt: time.Now()})
	slog.Info("ai router: cache initialised", "mode", mode)
}

// StartCacheRefresh starts a background goroutine that refreshes the mode cache every 3 seconds.
// The goroutine stops when ctx is cancelled.
func (s *Service) StartCacheRefresh(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mode, err := s.repo.GetCurrentMode(ctx)
				if err != nil {
					slog.Warn("ai router: cache refresh failed", "err", err)
					continue
				}
				s.modeCache.Store(&cachedMode{mode: mode, fetchedAt: time.Now()})
			}
		}
	}()
}

// GetCurrentConfig returns the current router_configs row for the GET /ai/config handler.
func (s *Service) GetCurrentConfig(ctx context.Context) (*RouterConfig, error) {
	return s.repo.GetCurrentConfig(ctx)
}

// SetMode inserts a new router_configs row for the requested mode.
// Returns ErrModeUnchanged if the mode is already active.
func (s *Service) SetMode(ctx context.Context, mode string, callerID uuid.UUID) (*RouterConfig, error) {
	current, err := s.repo.GetCurrentMode(ctx)
	if err != nil {
		return nil, fmt.Errorf("read current mode: %w", err)
	}
	if current == mode {
		return nil, ErrModeUnchanged
	}

	cfg, err := s.repo.InsertModeChange(ctx, mode, &current, &callerID)
	if err != nil {
		return nil, fmt.Errorf("insert mode change: %w", err)
	}
	// Update cache immediately — don't wait for the 3-second poll.
	s.modeCache.Store(&cachedMode{mode: mode, fetchedAt: time.Now()})
	slog.Info("ai router: mode changed", "prev", current, "new", mode, "by", callerID)
	return cfg, nil
}

// Dispatch routes a completion request to the appropriate backend based on the cached mode.
// It always writes an audit log entry unless messages validation failed before calling Dispatch.
func (s *Service) Dispatch(ctx context.Context, req DispatchRequest, primaryTimeoutSecs int) (*ChatResponse, *AIRequestLog, error) {
	mode := s.modeCache.Load().mode
	start := time.Now()

	entry := &AIRequestLog{
		RequestID: req.RequestID,
		Mode:      mode,
	}

	chatReq := req.toChatRequest(s.defaultModel)

	switch mode {
	case "local":
		return s.dispatchLocal(ctx, chatReq, entry, start, primaryTimeoutSecs)
	case "cloud":
		return s.dispatchCloud(ctx, chatReq, entry, start)
	case "auto":
		return s.dispatchAuto(ctx, chatReq, entry, start, primaryTimeoutSecs)
	default:
		return s.dispatchLocal(ctx, chatReq, entry, start, primaryTimeoutSecs)
	}
}

func (s *Service) dispatchLocal(ctx context.Context, req ChatRequest, entry *AIRequestLog, start time.Time, timeoutSecs int) (*ChatResponse, *AIRequestLog, error) {
	entry.BackendUsed = "local"
	pCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()

	resp, err := s.primaryClient.Complete(pCtx, req)
	entry.LatencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		entry.Status = "failed"
		entry.ErrorDetail = ptr(err.Error())
		_ = s.repo.LogRequest(context.Background(), entry)
		return nil, entry, err
	}
	entry.Status = "success"
	_ = s.repo.LogRequest(context.Background(), entry)
	return resp, entry, nil
}

func (s *Service) dispatchCloud(ctx context.Context, req ChatRequest, entry *AIRequestLog, start time.Time) (*ChatResponse, *AIRequestLog, error) {
	entry.BackendUsed = "cloud"
	resp, err := s.cloudClient.Complete(ctx, req)
	entry.LatencyMs = int(time.Since(start).Milliseconds())
	if err != nil {
		entry.Status = "failed"
		entry.ErrorDetail = ptr(err.Error())
		_ = s.repo.LogRequest(context.Background(), entry)
		return nil, entry, err
	}
	entry.Status = "success"
	_ = s.repo.LogRequest(context.Background(), entry)
	return resp, entry, nil
}

func (s *Service) dispatchAuto(ctx context.Context, req ChatRequest, entry *AIRequestLog, start time.Time, timeoutSecs int) (*ChatResponse, *AIRequestLog, error) {
	// Try primary first with its own timeout.
	pCtx, pCancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer pCancel()

	resp, pErr := s.primaryClient.Complete(pCtx, req)
	if pErr == nil {
		entry.BackendUsed = "local"
		entry.LatencyMs = int(time.Since(start).Milliseconds())
		entry.Status = "success"
		_ = s.repo.LogRequest(context.Background(), entry)
		return resp, entry, nil
	}

	// Primary failed — fall back to cloud.
	entry.FallbackUsed = true
	entry.FallbackReason = ptr(classifyPrimaryError(pErr))

	resp, cErr := s.cloudClient.Complete(ctx, req)
	entry.LatencyMs = int(time.Since(start).Milliseconds())
	if cErr != nil {
		// Both backends failed.
		entry.BackendUsed = "cloud"
		entry.Status = "failed"
		entry.ErrorDetail = ptr(fmt.Sprintf("primary: %v; cloud: %v", pErr, cErr))
		_ = s.repo.LogRequest(context.Background(), entry)
		return nil, entry, errors.New("all backends failed")
	}

	entry.BackendUsed = "cloud"
	entry.Status = "success"
	_ = s.repo.LogRequest(context.Background(), entry)
	return resp, entry, nil
}

// classifyPrimaryError maps a primary backend error to a fallback_reason string.
func classifyPrimaryError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "primary_timeout"
	}
	return "primary_error"
}
