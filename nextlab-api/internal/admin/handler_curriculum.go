package admin

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/Elder2003/nextlab-api/internal/rag"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func ListCurriculumDocsHandler(curRepo *rag.CurricularRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		carrera := c.Query("carrera", "")
		codigoMateria := c.Query("codigo_materia", "")
		semestre, _ := strconv.Atoi(c.Query("semestre", "0"))
		page, _ := strconv.Atoi(c.Query("page", "1"))
		limit, _ := strconv.Atoi(c.Query("limit", "20"))

		docs, total, err := curRepo.ListDocuments(c.Context(), carrera, semestre, codigoMateria, page, limit)
		if err != nil {
			slog.ErrorContext(c.Context(), "list curriculum docs error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.JSON(fiber.Map{
			"documents": docs,
			"total":     total,
		})
	}
}

func DeleteCurriculumDocHandler(curRepo *rag.CurricularRepository, mc *minio.Client, bucket string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid document id"})
		}

		minioPath, err := curRepo.DeleteDocument(c.Context(), id)
		if err != nil {
			if strings.Contains(err.Error(), "not_found") {
				return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "document not found"})
			}
			slog.ErrorContext(c.Context(), "delete curriculum doc error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		if minioPath != "" {
			if err := mc.RemoveObject(c.Context(), bucket, minioPath, minio.RemoveObjectOptions{}); err != nil {
				slog.ErrorContext(c.Context(), "remove minio object error", "path", minioPath, "err", err)
			}
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}
