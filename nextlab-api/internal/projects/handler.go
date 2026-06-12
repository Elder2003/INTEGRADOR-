package projects

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

func correlationID(c *fiber.Ctx) string {
	if v, ok := c.Locals("correlation_id").(string); ok {
		return v
	}
	return ""
}

func callerID(c *fiber.Ctx) (uuid.UUID, error) {
	return uuid.Parse(c.Locals("user_id").(string))
}

func callerRole(c *fiber.Ctx) string {
	r, _ := c.Locals("user_role").(string)
	return r
}

func errResp(c *fiber.Ctx, status int, code, msg string) error {
	return c.Status(status).JSON(fiber.Map{
		"error": code, "message": msg, "correlation_id": correlationID(c),
	})
}

func unauthorized(c *fiber.Ctx) error {
	return errResp(c, fiber.StatusUnauthorized, "unauthorized", "missing or invalid token")
}

func mapError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrForbidden):
		return errResp(c, fiber.StatusForbidden, "forbidden", "access denied")
	case errors.Is(err, ErrNotFound):
		return errResp(c, fiber.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, ErrConflict):
		return errResp(c, fiber.StatusConflict, "conflict", "name already in use")
	case errors.Is(err, ErrStageHasSubmissions):
		return errResp(c, fiber.StatusConflict, "stage_has_submissions", "cannot delete a stage that has submissions")
	case errors.Is(err, ErrNoStages):
		return errResp(c, fiber.StatusUnprocessableEntity, "no_stages", "investigacion activity must have at least one stage")
	case errors.Is(err, ErrWrongTipo):
		return errResp(c, fiber.StatusBadRequest, "wrong_activity_type", "this operation is not valid for this activity type")
	default:
		slog.ErrorContext(c.Context(), "internal error", "err", err, "correlation_id", correlationID(c))
		return errResp(c, fiber.StatusInternalServerError, "internal_error", "unexpected error")
	}
}

// ─── US1: Investigacion Stage Handlers ───────────────────────────────────────

func CreateStageHandler(svc *InvestigacionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}

		var req struct {
			Name         string    `json:"name"`
			DisplayOrder int       `json:"display_order"`
			Deadline     time.Time `json:"deadline"`
			Description  string    `json:"description"`
		}
		if err := c.BodyParser(&req); err != nil || req.Name == "" || req.Deadline.IsZero() {
			return errResp(c, fiber.StatusBadRequest, "validation_error", "name and deadline are required")
		}

		stage, err := svc.CreateStage(c.Context(), cid, aid, req.Name, req.DisplayOrder, req.Deadline, req.Description)
		if err != nil {
			return mapError(c, err)
		}
		return c.Status(fiber.StatusCreated).JSON(stage)
	}
}

func ListStagesHandler(svc *InvestigacionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		stages, err := svc.ListStages(c.Context(), aid)
		if err != nil {
			return mapError(c, err)
		}
		return c.JSON(fiber.Map{"stages": stages})
	}
}

func UpdateStageHandler(svc *InvestigacionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		stageID, err := uuid.Parse(c.Params("stageId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid stage id")
		}

		var req struct {
			Name         *string    `json:"name"`
			DisplayOrder *int       `json:"display_order"`
			Deadline     *time.Time `json:"deadline"`
			Description  *string    `json:"description"`
		}
		if err := c.BodyParser(&req); err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_request", "invalid body")
		}

		stage, err := svc.UpdateStage(c.Context(), cid, aid, stageID, req.Name, req.DisplayOrder, req.Deadline, req.Description)
		if err != nil {
			return mapError(c, err)
		}
		return c.JSON(stage)
	}
}

func DeleteStageHandler(svc *InvestigacionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		stageID, err := uuid.Parse(c.Params("stageId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid stage id")
		}

		if err := svc.DeleteStage(c.Context(), cid, aid, stageID); err != nil {
			return mapError(c, err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func SubmitToStageHandler(svc *InvestigacionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		stageID, err := uuid.Parse(c.Params("stageId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid stage id")
		}

		header, err := c.FormFile("file")
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "missing_file", "multipart field 'file' is required")
		}

		sub, err := svc.SubmitToStage(c.Context(), cid, aid, stageID, header)
		if err != nil {
			return mapError(c, err)
		}
		slog.InfoContext(c.Context(), "stage submission uploaded",
			"event", "stage_submission_uploaded", "student_id", cid, "stage_id", stageID, "is_late", sub.IsLate)
		return c.Status(fiber.StatusCreated).JSON(sub)
	}
}

func GetStageProgressHandler(svc *InvestigacionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}

		resp, err := svc.GetStageProgress(c.Context(), cid, aid)
		if err != nil {
			return mapError(c, err)
		}
		return c.JSON(resp)
	}
}

func GradeStageSubmissionHandler(svc *InvestigacionService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		stageID, err := uuid.Parse(c.Params("stageId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid stage id")
		}
		studentID, err := uuid.Parse(c.Params("studentId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid student id")
		}

		var req struct {
			Grade     float64 `json:"grade"`
			GradeNote string  `json:"grade_note"`
		}
		if err := c.BodyParser(&req); err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_request", "invalid body")
		}

		sub, err := svc.GradeStageSubmission(c.Context(), cid, aid, stageID, studentID, req.Grade, req.GradeNote)
		if err != nil {
			if err.Error() == "grade must be 0–100" {
				return errResp(c, fiber.StatusBadRequest, "validation_error", err.Error())
			}
			return mapError(c, err)
		}
		return c.JSON(sub)
	}
}

// ─── US1: Publication Guard ───────────────────────────────────────────────────

// ValidatePublishHandler rejects publication of an investigacion activity with zero stages (FR-007).
func ValidatePublishHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}

		tipo, cursoID, _, err := repo.GetActividadTipo(c.Context(), aid)
		if err != nil {
			return mapError(c, err)
		}
		docenteID, err := repo.GetCursoDocente(c.Context(), cursoID)
		if err != nil {
			return mapError(c, err)
		}
		if docenteID != cid {
			return errResp(c, fiber.StatusForbidden, "forbidden", "access denied")
		}

		if tipo == "investigacion" {
			n, err := repo.CountStagesForActividad(c.Context(), aid)
			if err != nil {
				return mapError(c, err)
			}
			if n == 0 {
				return errResp(c, fiber.StatusUnprocessableEntity, "no_stages",
					"investigacion activity must have at least one stage before publishing")
			}
		}
		return c.JSON(fiber.Map{"status": "ok", "actividad_id": aid})
	}
}

// ─── US2: Git Analysis Handlers ───────────────────────────────────────────────

func GitSubmitHandler(svc *GitAnalysisService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}

		var req struct {
			RepoURL string `json:"repo_url"`
		}
		if err := c.BodyParser(&req); err != nil || req.RepoURL == "" {
			return errResp(c, fiber.StatusBadRequest, "validation_error", "repo_url is required")
		}

		sub, analysisStatus, err := svc.SubmitGitURL(c.Context(), cid, aid, req.RepoURL)
		if err != nil {
			switch err.Error() {
			case "host_not_allowed":
				return errResp(c, fiber.StatusUnprocessableEntity, "host_not_allowed",
					"Only github.com, gitlab.com, bitbucket.org are accepted")
			case "scheme_not_allowed":
				return errResp(c, fiber.StatusUnprocessableEntity, "scheme_not_allowed",
					"Only https:// URLs are accepted")
			}
			return mapError(c, err)
		}
		slog.InfoContext(c.Context(), "git url submitted", "event", "git_submitted", "student_id", cid, "actividad_id", aid)
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
			"git_submission_id": sub.ID,
			"actividad_id":      sub.ActividadID,
			"student_id":        sub.StudentID,
			"repo_url":          sub.RepoURL,
			"submitted_at":      sub.SubmittedAt,
			"analysis_status":   analysisStatus,
		})
	}
}

func GetGitReportHandler(svc *GitAnalysisService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		role := callerRole(c)
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		studentID, err := uuid.Parse(c.Params("studentId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid student id")
		}

		report, err := svc.GetGitReport(c.Context(), cid, role, aid, studentID)
		if err != nil {
			return mapError(c, err)
		}
		return c.JSON(report)
	}
}

func GitGradeHandler(svc *GitAnalysisService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		studentID, err := uuid.Parse(c.Params("studentId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid student id")
		}

		var req struct {
			Grade          float64 `json:"grade"`
			OverrideReason string  `json:"override_reason"`
		}
		if err := c.BodyParser(&req); err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_request", "invalid body")
		}

		analysis, err := svc.GradeGitAnalysis(c.Context(), cid, aid, studentID, req.Grade, req.OverrideReason)
		if err != nil {
			if err.Error() == "grade must be 0–100" {
				return errResp(c, fiber.StatusBadRequest, "validation_error", err.Error())
			}
			return mapError(c, err)
		}
		return c.JSON(fiber.Map{
			"git_submission_id": analysis.GitSubmissionID,
			"student_id":        studentID,
			"grade":             analysis.Grade,
			"override_reason":   analysis.OverrideReason,
			"graded_at":         analysis.GradedAt,
		})
	}
}

// ─── US3: Oral Defense Handlers ───────────────────────────────────────────────

func ScheduleDefenseHandler(svc *DefensaService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		studentID, err := uuid.Parse(c.Params("studentId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid student id")
		}

		var req struct {
			ScheduledAt time.Time `json:"scheduled_at"`
		}
		if err := c.BodyParser(&req); err != nil || req.ScheduledAt.IsZero() {
			return errResp(c, fiber.StatusBadRequest, "validation_error", "scheduled_at is required")
		}

		appt, err := svc.ScheduleDefense(c.Context(), cid, aid, studentID, req.ScheduledAt)
		if err != nil {
			return mapError(c, err)
		}
		return c.JSON(appt)
	}
}

func GetDefenseAppointmentHandler(svc *DefensaService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		role := callerRole(c)
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		studentID, err := uuid.Parse(c.Params("studentId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid student id")
		}

		appt, err := svc.GetDefenseAppointment(c.Context(), cid, role, aid, studentID)
		if err != nil {
			return mapError(c, err)
		}
		if appt == nil {
			return c.JSON(fiber.Map{
				"actividad_id": aid,
				"student_id":   studentID,
				"scheduled_at": nil,
				"grade":        nil,
			})
		}
		return c.JSON(appt)
	}
}

func GradeDefenseHandler(svc *DefensaService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cid, err := callerID(c)
		if err != nil {
			return unauthorized(c)
		}
		aid, err := uuid.Parse(c.Params("actividadId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid actividad id")
		}
		studentID, err := uuid.Parse(c.Params("studentId"))
		if err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_id", "invalid student id")
		}

		var req struct {
			Grade float64 `json:"grade"`
		}
		if err := c.BodyParser(&req); err != nil {
			return errResp(c, fiber.StatusBadRequest, "invalid_request", "invalid body")
		}

		appt, err := svc.GradeDefense(c.Context(), cid, aid, studentID, req.Grade)
		if err != nil {
			if err.Error() == "grade must be 0–100" {
				return errResp(c, fiber.StatusBadRequest, "validation_error", err.Error())
			}
			return mapError(c, err)
		}
		return c.JSON(fiber.Map{
			"actividad_id": appt.ActividadID,
			"student_id":   appt.StudentID,
			"grade":        appt.Grade,
			"graded_at":    appt.GradedAt,
		})
	}
}
