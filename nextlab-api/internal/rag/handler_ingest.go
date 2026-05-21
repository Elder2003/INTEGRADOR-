package rag

import (
	"io"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

var allowedExts = map[string]bool{".pdf": true, ".docx": true, ".txt": true}

func ingestCurricularHandler(svc *IngestService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "multipart form required"})
		}

		carrera := formVal(form, "carrera")
		semestreStr := formVal(form, "semestre")
		codigoMateria := formVal(form, "codigo_materia")

		if carrera == "" || semestreStr == "" || codigoMateria == "" {
			return c.Status(400).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "carrera, semestre, and codigo_materia are required",
			})
		}

		semestre := parseInt(semestreStr)
		if semestre < 1 || semestre > 10 {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "semestre must be between 1 and 10"})
		}

		isRubric := formVal(form, "is_rubric") == "true"
		documentID := parseOptionalUUID(formVal(form, "document_id"))
		actividadID := parseOptionalUUID(formVal(form, "actividad_id"))

		fileData, filename, err := readUploadedFile(form)
		if err != nil {
			if err.Error() == "unsupported file type" {
				return c.Status(422).JSON(fiber.Map{
					"error":   "unsupported_file_type",
					"message": "only PDF, DOCX, and TXT files are accepted",
				})
			}
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": err.Error()})
		}

		externalURL := formVal(form, "external_url")

		result, err := svc.IngestCurricular(c.Context(), IngestCurricularRequest{
			FileData:         fileData,
			FileSize:         int64(len(fileData)),
			OriginalFilename: filename,
			Carrera:          carrera,
			Semestre:         semestre,
			CodigoMateria:    codigoMateria,
			IsRubric:         isRubric,
			DocumentID:       documentID,
			ActividadID:      actividadID,
			ExternalURL:      externalURL,
		})
		if err != nil {
			if err.Error() == "unsupported file type" {
				return c.Status(422).JSON(fiber.Map{
					"error":   "unsupported_file_type",
					"message": "only PDF, DOCX, and TXT files are accepted",
				})
			}
			if err.Error() == "embedding_failed" {
				return c.Status(500).JSON(fiber.Map{
					"error":   "embedding_failed",
					"message": "embedding service unavailable; document saved but not indexed",
				})
			}
			slog.Error("curricular ingest failed", "err", err)
			return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": err.Error()})
		}

		status := 201
		if documentID != nil {
			status = 200
		}
		return c.Status(status).JSON(result)
	}
}

func ingestDocenteHandler(svc *IngestService, docRepo *DocenteRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, ok := userIDFromLocals(c)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}

		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "multipart form required"})
		}

		cursoIDStr := formVal(form, "curso_id")
		if cursoIDStr == "" {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "curso_id is required"})
		}
		cursoID, err := uuid.Parse(cursoIDStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "curso_id must be a valid UUID"})
		}

		// Verify ownership before any ingest work
		owns, err := docRepo.VerifyDocenteOwnsCourse(c.Context(), docenteID, cursoID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "internal_error"})
		}
		if !owns {
			return c.Status(403).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "you do not have access to this course",
			})
		}

		isRubric := formVal(form, "is_rubric") == "true"
		documentID := parseOptionalUUID(formVal(form, "document_id"))
		actividadID := parseOptionalUUID(formVal(form, "actividad_id"))

		fileData, filename, err := readUploadedFile(form)
		if err != nil {
			if err.Error() == "unsupported file type" {
				return c.Status(422).JSON(fiber.Map{
					"error":   "unsupported_file_type",
					"message": "only PDF, DOCX, and TXT files are accepted",
				})
			}
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": err.Error()})
		}

		externalURL := formVal(form, "external_url")

		result, err := svc.IngestDocente(c.Context(), IngestDocenteRequest{
			FileData:         fileData,
			FileSize:         int64(len(fileData)),
			OriginalFilename: filename,
			DocenteID:        docenteID,
			CursoID:          cursoID,
			ActividadID:      actividadID,
			IsRubric:         isRubric,
			DocumentID:       documentID,
			ExternalURL:      externalURL,
		})
		if err != nil {
			if err.Error() == "unsupported file type" {
				return c.Status(422).JSON(fiber.Map{
					"error":   "unsupported_file_type",
					"message": "only PDF, DOCX, and TXT files are accepted",
				})
			}
			if err.Error() == "embedding_failed" {
				return c.Status(500).JSON(fiber.Map{
					"error":   "embedding_failed",
					"message": "embedding service unavailable; document saved but not indexed",
				})
			}
			slog.Error("docente ingest failed", "err", err)
			return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": err.Error()})
		}

		status := 201
		if documentID != nil {
			status = 200
		}
		return c.Status(status).JSON(result)
	}
}

// --- helpers ---

func formVal(form *multipart.Form, key string) string {
	if vals, ok := form.Value[key]; ok && len(vals) > 0 {
		return strings.TrimSpace(vals[0])
	}
	return ""
}

func readUploadedFile(form *multipart.Form) ([]byte, string, error) {
	files, ok := form.File["file"]
	if !ok || len(files) == 0 {
		return nil, "", fiber.NewError(400, "file is required")
	}
	fh := files[0]
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if !allowedExts[ext] {
		return nil, "", fiber.NewError(422, "unsupported file type")
	}
	f, err := fh.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	return data, fh.Filename, err
}

func parseOptionalUUID(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

func parseInt(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}

func userIDFromLocals(c *fiber.Ctx) (uuid.UUID, bool) {
	raw := c.Locals("user_id")
	if raw == nil {
		return uuid.UUID{}, false
	}
	switch v := raw.(type) {
	case uuid.UUID:
		return v, true
	case string:
		id, err := uuid.Parse(v)
		return id, err == nil
	}
	return uuid.UUID{}, false
}
