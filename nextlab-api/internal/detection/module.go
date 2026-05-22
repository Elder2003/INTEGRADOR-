package detection

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// WorkerHandler runs the two-phase detection pipeline for a completed exam session.
type WorkerHandler struct {
	svc *DetectionService
}

func NewWorkerHandler(svc *DetectionService) *WorkerHandler {
	return &WorkerHandler{svc: svc}
}

// Run executes heuristic scoring then LLM analysis for the given session.
// Both phases are idempotent and isolated; LLM failure is stored, not propagated.
func (w *WorkerHandler) Run(ctx context.Context, sessionID uuid.UUID) error {
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
