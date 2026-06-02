package baseline

import (
	"context"
	"log/slog"
	"math"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetBaseline fetches the baseline for a student (nil if none).
func (s *Service) GetBaseline(ctx context.Context, studentID uuid.UUID) (*StudentBehaviorBaseline, error) {
	return s.repo.Get(ctx, studentID)
}

// ComputeZScore returns z-scores for tab_blurs and clipboard rates vs the student's baseline.
// Returns (0, 0, false) when fewer than 3 sessions have been analyzed.
func (s *Service) ComputeZScore(b *StudentBehaviorBaseline, tabBlursPerHour, clipboardPerHour float64) (zTab, zClip float64, available bool) {
	if b == nil || b.TotalSessionsAnalyzed < 3 {
		return 0, 0, false
	}
	zTab = (tabBlursPerHour - b.AvgTabBlursPerHour) / math.Max(b.StddevTabBlurs, 0.01)
	zClip = (clipboardPerHour - b.AvgClipboardPerHour) / math.Max(b.StddevClipboard, 0.01)
	return zTab, zClip, true
}

// UpdateAfterAnalysis updates the rolling 10-session baseline for the student.
// If isFraud is true the session is excluded to prevent contaminating the baseline.
func (s *Service) UpdateAfterAnalysis(ctx context.Context, studentID uuid.UUID, tabBlursPerHour, clipboardPerHour float64, isFraud bool) error {
	if isFraud {
		return nil
	}

	b, err := s.repo.Get(ctx, studentID)
	if err != nil {
		return err
	}
	if b == nil {
		b = &StudentBehaviorBaseline{StudentID: studentID}
	}

	// Append new values and keep last 10
	b.Last10TabBlurs = append(b.Last10TabBlurs, tabBlursPerHour)
	b.Last10Clipboard = append(b.Last10Clipboard, clipboardPerHour)
	if len(b.Last10TabBlurs) > 10 {
		b.Last10TabBlurs = b.Last10TabBlurs[len(b.Last10TabBlurs)-10:]
	}
	if len(b.Last10Clipboard) > 10 {
		b.Last10Clipboard = b.Last10Clipboard[len(b.Last10Clipboard)-10:]
	}

	b.AvgTabBlursPerHour, b.StddevTabBlurs = calcMeanStddev(b.Last10TabBlurs)
	b.AvgClipboardPerHour, b.StddevClipboard = calcMeanStddev(b.Last10Clipboard)
	b.TotalSessionsAnalyzed++

	if err := s.repo.Upsert(ctx, b); err != nil {
		slog.Error("baseline: upsert failed", "student_id", studentID, "err", err)
		return err
	}
	return nil
}

func calcMeanStddev(values []float64) (mean, stddev float64) {
	if len(values) == 0 {
		return 0, 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))
	var varianceSum float64
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}
	stddev = math.Sqrt(varianceSum / float64(len(values)))
	return mean, stddev
}
