package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/Elder2003/nextlab-api/internal/ai"
	"github.com/Elder2003/nextlab-api/internal/config"
	"github.com/Elder2003/nextlab-api/internal/rag"
)

type DetectionService struct {
	repo        *DetectionRepository
	heuristic   *HeuristicEngine
	aiSvc       *ai.Service
	ragSvc      *rag.RetrieveService
	minio       *minio.Client
	minioBucket string
	cfg         *config.Config
}

func NewDetectionService(
	repo *DetectionRepository,
	heuristic *HeuristicEngine,
	aiSvc *ai.Service,
	ragSvc *rag.RetrieveService,
	minioClient *minio.Client,
	minioBucket string,
	cfg *config.Config,
) *DetectionService {
	return &DetectionService{
		repo:        repo,
		heuristic:   heuristic,
		aiSvc:       aiSvc,
		ragSvc:      ragSvc,
		minio:       minioClient,
		minioBucket: minioBucket,
		cfg:         cfg,
	}
}

// RunHeuristic scores the session's trace events and stores a 'heuristic' row.
// Idempotent: returns nil immediately if a heuristic row already exists (FR-012).
func (s *DetectionService) RunHeuristic(ctx context.Context, sessionID uuid.UUID) error {
	exists, err := s.repo.HeuristicExists(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("check heuristic exists: %w", err)
	}
	if exists {
		return nil
	}

	events, err := s.repo.GetTraceEvents(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get trace events: %w", err)
	}

	score, flagged := s.heuristic.Score(events)

	if _, err := s.repo.InsertHeuristic(ctx, sessionID, score, flagged, len(events)); err != nil {
		return fmt.Errorf("insert heuristic: %w", err)
	}

	slog.InfoContext(ctx, "heuristic analysis stored",
		"session_id", sessionID,
		"score", score,
		"flagged", flagged,
		"event_count", len(events),
	)
	return nil
}

// RunLLM builds a RAG-enriched prompt and calls the AI router, storing the 'llm' row.
// Idempotent: returns nil if an LLM row already exists (FR-012).
// On AI/prompt failure, stores llm_status='failed' and returns nil (FR-010, Principle II).
func (s *DetectionService) RunLLM(ctx context.Context, sessionID uuid.UUID) error {
	exists, err := s.repo.LLMExists(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("check llm exists: %w", err)
	}
	if exists {
		return nil
	}

	actividadID, submissionID, _, err := s.repo.GetSessionForDetection(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	// --- Retrieve submission text ---
	submissionText := ""
	if submissionID != uuid.Nil {
		storageKey, originalFilename, err := s.repo.GetSubmissionStorageKey(ctx, submissionID)
		if err != nil {
			slog.WarnContext(ctx, "could not get submission storage key", "submission_id", submissionID, "err", err)
		} else {
			obj, err := s.minio.GetObject(ctx, s.minioBucket, storageKey, minio.GetObjectOptions{})
			if err != nil {
				slog.WarnContext(ctx, "could not fetch submission from minio", "key", storageKey, "err", err)
			} else {
				defer obj.Close()
				data, err := io.ReadAll(obj)
				if err != nil {
					slog.WarnContext(ctx, "could not read submission bytes", "err", err)
				} else {
					ext := strings.ToLower(filepath.Ext(originalFilename))
					extracted, err := rag.ExtractText(data, ext)
					if err != nil {
						slog.WarnContext(ctx, "text extraction failed", "ext", ext, "err", err)
					} else {
						submissionText = extracted
					}
				}
			}
		}
	}

	// --- Curricular RAG ---
	var curricularResp *rag.RetrievalResponse
	carrera, codigoMateria, semestre, err := s.repo.GetCurricularScope(ctx, actividadID)
	if err != nil {
		slog.WarnContext(ctx, "could not get curricular scope", "actividad_id", actividadID, "err", err)
	} else {
		resp, err := s.ragSvc.QueryCurricular(ctx, rag.CurricularQueryRequest{
			Carrera:       carrera,
			Semestre:      semestre,
			CodigoMateria: codigoMateria,
			Query:         submissionText,
			ActividadID:   &actividadID,
		})
		if err != nil {
			slog.WarnContext(ctx, "curricular RAG query failed", "err", err)
		} else {
			curricularResp = resp
		}
	}

	// --- Docente RAG ---
	var docenteResp *rag.RetrievalResponse
	docenteID, cursoID, err := s.repo.GetDocenteScope(ctx, actividadID)
	if err != nil {
		slog.WarnContext(ctx, "could not get docente scope", "actividad_id", actividadID, "err", err)
	} else {
		resp, err := s.ragSvc.QueryDocente(ctx, rag.DocenteQueryRequest{
			DocenteID:   docenteID,
			CursoID:     cursoID,
			ActividadID: &actividadID,
			Query:       submissionText,
		})
		if err != nil {
			slog.WarnContext(ctx, "docente RAG query failed", "err", err)
		} else {
			docenteResp = resp
		}
	}

	// --- Fetch heuristic result for prompt ---
	heuristic, _, err := s.repo.GetAnalysisResults(ctx, sessionID)
	if err != nil {
		slog.WarnContext(ctx, "could not fetch heuristic result for prompt", "err", err)
	}

	// --- Build prompt ---
	systemMsg := "You are an academic integrity assistant. Analyse the student submission below and provide an advisory assessment. Do NOT assign a final grade — your output is advisory only and will be reviewed by the course instructor."
	userMsg := buildPrompt(submissionText, heuristic, curricularResp, docenteResp)

	// --- Dispatch to AI router ---
	chatResp, aiLog, err := s.aiSvc.Dispatch(ctx, ai.DispatchRequest{
		Messages: []ai.ChatMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: userMsg},
		},
	}, s.cfg.AIPrimaryTimeoutSecs)

	if err != nil || chatResp == nil {
		slog.WarnContext(ctx, "LLM dispatch failed; storing failed result", "session_id", sessionID, "err", err)
		backend := ""
		if aiLog != nil {
			backend = aiLog.BackendUsed
		}
		_, insertErr := s.repo.InsertLLMResult(ctx, sessionID, "", "", backend, "failed")
		if insertErr != nil {
			return fmt.Errorf("insert failed llm result: %w", insertErr)
		}
		return nil
	}

	// --- Parse JSON response ---
	rawContent := ""
	if len(chatResp.Choices) > 0 {
		rawContent = chatResp.Choices[0].Message.Content
	}

	advisoryText := rawContent
	suggestedRange := "unknown"

	var parsed struct {
		AdvisoryText        string `json:"advisory_text"`
		SuggestedGradeRange string `json:"suggested_grade_range"`
	}
	if err := json.Unmarshal([]byte(rawContent), &parsed); err == nil {
		if parsed.AdvisoryText != "" {
			advisoryText = parsed.AdvisoryText
		}
		if parsed.SuggestedGradeRange != "" {
			suggestedRange = parsed.SuggestedGradeRange
		}
	}

	backend := ""
	if aiLog != nil {
		backend = aiLog.BackendUsed
	}

	if _, err := s.repo.InsertLLMResult(ctx, sessionID, advisoryText, suggestedRange, backend, "success"); err != nil {
		return fmt.Errorf("insert llm result: %w", err)
	}

	slog.InfoContext(ctx, "llm analysis stored",
		"session_id", sessionID,
		"suggested_range", suggestedRange,
		"backend", backend,
	)
	return nil
}

// GetReport returns the detection report for a session, scoped to the docente.
func (s *DetectionService) GetReport(ctx context.Context, sessionID, docenteID uuid.UUID) (*DetectionReport, error) {
	owns, err := s.repo.VerifyDocenteOwnsSession(ctx, sessionID, docenteID)
	if err != nil {
		return nil, fmt.Errorf("verify ownership: %w", err)
	}
	if !owns {
		return nil, ErrForbidden
	}

	actividadID, submissionID, studentID, err := s.repo.GetSessionForDetection(ctx, sessionID)
	if err != nil {
		return nil, ErrNotFound
	}

	heuristic, llm, err := s.repo.GetAnalysisResults(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get analysis results: %w", err)
	}
	if heuristic == nil {
		return nil, ErrNotFound
	}

	gradeDecision, err := s.repo.GetGradeDecision(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get grade decision: %w", err)
	}

	formatReport, err := s.repo.GetFormatReport(ctx, sessionID, submissionID)
	if err != nil {
		slog.WarnContext(ctx, "could not fetch format report", "session_id", sessionID, "err", err)
	}

	llmStatus := "unavailable"
	if llm != nil {
		llmStatus = llm.LLMStatus
	}

	return &DetectionReport{
		SessionID:     sessionID,
		StudentID:     studentID,
		ActividadID:   actividadID,
		FormatReport:  formatReport,
		Heuristic:     heuristic,
		LLM:           llm,
		LLMStatus:     llmStatus,
		GradeDecision: gradeDecision,
	}, nil
}

// SubmitGrade stores a grade decision. Validates override reason when grade outside suggested range.
func (s *DetectionService) SubmitGrade(ctx context.Context, sessionID, docenteID uuid.UUID, req GradeRequest) (*GradeDecisionResult, error) {
	owns, err := s.repo.VerifyDocenteOwnsSession(ctx, sessionID, docenteID)
	if err != nil {
		return nil, fmt.Errorf("verify ownership: %w", err)
	}
	if !owns {
		return nil, ErrForbidden
	}

	exists, err := s.repo.GradeDecisionExists(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("check grade exists: %w", err)
	}
	if exists {
		return nil, ErrConflict
	}

	_, llm, err := s.repo.GetAnalysisResults(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get analysis results: %w", err)
	}
	if llm == nil {
		return nil, ErrNotFound
	}

	// Validate override reason requirement
	suggestedRange := llm.SuggestedGradeRange
	if requiresOverrideReason(req.FinalGrade, suggestedRange) && req.OverrideReason == "" {
		return nil, ErrOverrideRequired
	}

	if err := s.repo.InsertGradeDecision(ctx, sessionID, llm.ID, docenteID,
		req.FinalGrade, suggestedRange, req.OverrideReason); err != nil {
		return nil, fmt.Errorf("insert grade decision: %w", err)
	}

	g, err := s.repo.GetGradeDecision(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("fetch grade decision: %w", err)
	}
	return g, nil
}

// buildPrompt constructs the LLM user message from all available context.
func buildPrompt(submissionText string, h *HeuristicResult, curricular, docente *rag.RetrievalResponse) string {
	var sb strings.Builder

	sb.WriteString("## Submission Text\n")
	if submissionText == "" {
		sb.WriteString("(file content unavailable)\n")
	} else {
		sb.WriteString(submissionText)
		sb.WriteString("\n")
	}

	sb.WriteString("\n## Behavioral Analysis\n")
	if h != nil {
		sb.WriteString(fmt.Sprintf("Suspicion score: %d/100\n", h.Score))
		if len(h.FlaggedCategories) > 0 {
			sb.WriteString(fmt.Sprintf("Flagged indicators: %s\n", strings.Join(h.FlaggedCategories, ", ")))
		} else {
			sb.WriteString("Flagged indicators: none\n")
		}
		sb.WriteString(fmt.Sprintf("Total trace events analysed: %d\n", h.EventCount))
	} else {
		sb.WriteString("Behavioral analysis not available.\n")
	}

	sb.WriteString("\n## Curricular Context\n")
	if curricular != nil && len(curricular.Chunks) > 0 {
		for _, chunk := range curricular.Chunks {
			sb.WriteString(chunk.ChunkText)
			sb.WriteString("\n---\n")
		}
	} else {
		sb.WriteString("(no curricular context available)\n")
	}

	// Rubric always injected in full — never truncated (Constitution Principle V)
	sb.WriteString("\n## Rubric (full, untruncated)\n")
	hasRubric := false
	if curricular != nil {
		for _, r := range curricular.Rubrics {
			sb.WriteString(r.ChunkText)
			sb.WriteString("\n")
			hasRubric = true
		}
	}
	if docente != nil {
		for _, r := range docente.Rubrics {
			sb.WriteString(r.ChunkText)
			sb.WriteString("\n")
			hasRubric = true
		}
	}
	if !hasRubric {
		sb.WriteString("(no rubric available)\n")
	}

	sb.WriteString("\n## Task\n")
	sb.WriteString("1. Assess whether the submission is consistent with the student's expected knowledge level.\n")
	sb.WriteString("2. Suggest a grade range (e.g., \"60-70\" or \"Reprobado\") with a brief justification.\n")
	sb.WriteString("3. Note any specific concerns raised by the behavioral indicators.\n")
	sb.WriteString("Format your response as JSON: {\"advisory_text\":\"...\",\"suggested_grade_range\":\"...\"}\n")

	return sb.String()
}

// requiresOverrideReason returns true when finalGrade is outside the numeric range in suggestedRange.
// Uses a 5-point tolerance. Non-numeric ranges (e.g. "Reprobado") do not require a reason.
func requiresOverrideReason(finalGrade float64, suggestedRange string) bool {
	sep := ""
	if strings.Contains(suggestedRange, "–") {
		sep = "–"
	} else if strings.Contains(suggestedRange, "-") {
		sep = "-"
	} else {
		return false
	}

	parts := strings.SplitN(suggestedRange, sep, 2)
	if len(parts) != 2 {
		return false
	}

	var lo, hi float64
	if _, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%f", &lo); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &hi); err != nil {
		return false
	}

	const tolerance = 5.0
	return finalGrade < lo-tolerance || finalGrade > hi+tolerance
}
