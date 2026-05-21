package rag

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func retrieveCurricularHandler(svc *RetrieveService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req CurricularQueryRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "JSON body required"})
		}

		if req.Carrera == "" || req.Semestre == 0 || req.CodigoMateria == "" {
			return c.Status(400).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "carrera, semestre, and codigo_materia are required",
			})
		}
		if req.Query == "" {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "query is required"})
		}

		result, err := svc.QueryCurricular(c.Context(), req)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": err.Error()})
		}
		return c.Status(200).JSON(result)
	}
}

func retrieveDocenteHandler(svc *RetrieveService, docRepo *DocenteRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		docenteID, ok := userIDFromLocals(c)
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}

		var req DocenteQueryRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "JSON body required"})
		}

		if req.CursoID == (uuid.UUID{}) {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "curso_id is required"})
		}
		if req.Query == "" {
			return c.Status(400).JSON(fiber.Map{"error": "invalid_request", "message": "query is required"})
		}

		req.DocenteID = docenteID

		owns, err := docRepo.VerifyDocenteOwnsCourse(c.Context(), docenteID, req.CursoID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "internal_error"})
		}
		if !owns {
			return c.Status(403).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "you do not have access to this course",
			})
		}

		result, err := svc.QueryDocente(c.Context(), req)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": err.Error()})
		}
		return c.Status(200).JSON(result)
	}
}
