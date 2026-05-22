package detection

import (
	"context"
	"log/slog"

	"github.com/Elder2003/nextlab-api/internal/format"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// WorkerHandler runs the three-phase analysis pipeline for a completed exam session.
type WorkerHandler struct {
	fmtSvc *format.FormatService
	svc    *DetectionService
}

func NewWorkerHandler(fmtSvc *format.FormatService, svc *DetectionService) *WorkerHandler {
	return &WorkerHandler{fmtSvc: fmtSvc, svc: svc}
}

// Run executes format validation (phase 0), heuristic scoring (phase 1), and LLM analysis (phase 2).
// All phases are idempotent and isolated; LLM failure is stored, not propagated.
func (w *WorkerHandler) Run(ctx context.Context, sessionID uuid.UUID) error {
	// Phase 0: format validation — fastest, always deterministic
	if err := w.fmtSvc.Run(ctx, sessionID); err != nil {
		slog.WarnContext(ctx, "format validation phase failed", "session_id", sessionID, "err", err)
	}

	if err := w.svc.RunHeuristic(ctx, sessionID); err != nil {
		return err
	}
	if err := w.svc.RunLLM(ctx, sessionID); err != nil {
		slog.WarnContext(ctx, "llm phase failed", "session_id", sessionID, "err", err)
	}
	return nil
}

// RegisterRoutes wires detection endpoints under /sessions/:sessionID.
// Students are denied at the role guard — only docentes may access these routes.
func RegisterRoutes(
	app *fiber.App,
	svc *DetectionService,
	authMiddleware fiber.Handler,
	roleGuard func(...string) fiber.Handler,
) {
	h := newDetectionHandler(svc)

	sessions := app.Group("/sessions/:sessionID",
		authMiddleware,
		roleGuard("docente"),
	)

	sessions.Get("/detection-report", h.report)
	sessions.Post("/grade", h.grade)
}
