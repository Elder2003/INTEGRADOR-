package mldetection

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

type Service struct {
	repo     *Repository
	runtime  *ONNXRuntime
	modelDir string
	dbDSN    string
}

func NewService(repo *Repository, runtime *ONNXRuntime, modelDir, dbDSN string) *Service {
	return &Service{repo: repo, runtime: runtime, modelDir: modelDir, dbDSN: dbDSN}
}

// Retrain triggers the Python training script. Requires python3 + tools/ml/train.py.
func (s *Service) Retrain(ctx context.Context, minSessions int) error {
	count, err := s.repo.CountLabeledSessions(ctx)
	if err != nil {
		return fmt.Errorf("count sessions: %w", err)
	}
	if count < minSessions {
		return &InsufficientDataError{Required: minSessions, Available: count}
	}

	if err := os.MkdirAll(s.modelDir, 0o755); err != nil {
		return fmt.Errorf("create model dir: %w", err)
	}

	trainCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(trainCtx, "python3", "tools/ml/train.py",
		"--db-dsn", s.dbDSN,
		"--model-dir", s.modelDir,
		fmt.Sprintf("--min-sessions=%d", minSessions),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("training script failed: %w", err)
	}

	// Reload active model metadata from DB
	meta, err := s.repo.GetActive(ctx)
	if err != nil || meta == nil {
		slog.Warn("mldetection: trained but could not reload active model", "err", err)
		return nil
	}

	if s.runtime != nil {
		if err := s.runtime.HotSwap(meta.FilePath, meta.FeatureNames); err != nil {
			return fmt.Errorf("hot swap model: %w", err)
		}
	} else {
		s.runtime, _ = Load(meta.FilePath, meta.FeatureNames)
	}

	slog.Info("mldetection: model reloaded", "version", meta.Version, "file", meta.FilePath)
	return nil
}

// GetActiveMeta returns the metadata of the currently loaded model (nil if none).
func (s *Service) GetActiveMeta() *MLModel {
	if s.runtime == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	meta, _ := s.repo.GetActive(ctx)
	return meta
}

// GetRuntime returns the active ONNX runtime (nil if none loaded).
func (s *Service) GetRuntime() *ONNXRuntime {
	return s.runtime
}

// InsufficientDataError is returned when there are not enough labeled sessions.
type InsufficientDataError struct {
	Required  int
	Available int
}

func (e *InsufficientDataError) Error() string {
	return fmt.Sprintf("insufficient training data: need %d, have %d", e.Required, e.Available)
}
