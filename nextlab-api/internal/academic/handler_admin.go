package academic

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Carreras ─────────────────────────────────────────────────────────────────

func ListCarrerasHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		list, err := GetCarreras(c.Context(), db)
		if err != nil {
			return internalError(c)
		}
		if list == nil {
			list = []Carrera{}
		}
		return c.JSON(list)
	}
}

func CreateCarreraHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req CreateCarreraRequest
		if err := c.BodyParser(&req); err != nil || req.Code == "" || req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "validation_error",
				Message: "code and name are required",
			})
		}

		carrera, err := CreateCarrera(c.Context(), db, req.Code, req.Name)
		if err != nil {
			if isUniqueViolation(err) {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
					Error:   "conflict",
					Message: "a carrera with that code already exists",
				})
			}
			return internalError(c)
		}

		return c.Status(fiber.StatusCreated).JSON(carrera)
	}
}

func UpdateCarreraHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid carrera id"})
		}

		var req UpdateCarreraRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_request", Message: "invalid body"})
		}

		carrera, err := UpdateCarrera(c.Context(), db, id, req.Name, req.IsActive)
		if err != nil {
			return internalError(c)
		}
		if carrera == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "carrera not found"})
		}
		return c.JSON(carrera)
	}
}

func DeleteCarreraHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid carrera id"})
		}

		hasMaterias, err := CarreraHasActiveMaterias(c.Context(), db, id)
		if err != nil {
			return internalError(c)
		}
		if hasMaterias {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Error:   "conflict",
				Message: "carrera has active materias; deactivate them first",
			})
		}

		f := false
		_, err = UpdateCarrera(c.Context(), db, id, nil, &f)
		if err != nil {
			return internalError(c)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ─── Materias ─────────────────────────────────────────────────────────────────

func ListMateriasHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var carreraID *uuid.UUID
		if raw := c.Query("carrera_id"); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_param", Message: "invalid carrera_id"})
			}
			carreraID = &id
		}

		var semester *int
		if raw := c.Query("semester"); raw != "" {
			var s int
			if _, err := parseIntQuery(raw, &s); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_param", Message: "invalid semester"})
			}
			semester = &s
		}

		list, err := GetMaterias(c.Context(), db, carreraID, semester)
		if err != nil {
			return internalError(c)
		}
		if list == nil {
			list = []Materia{}
		}
		return c.JSON(list)
	}
}

func CreateMateriaHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req CreateMateriaRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_request", Message: "invalid body"})
		}
		if req.Code == "" || req.Name == "" || req.Semester < 1 || req.Semester > 10 || req.CarreraID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "validation_error",
				Message: "carrera_id, code, name, and semester (1-10) are required",
			})
		}

		materia, err := CreateMateria(c.Context(), db, req.CarreraID, req.Code, req.Name, req.Semester)
		if err != nil {
			if isUniqueViolation(err) {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
					Error:   "conflict",
					Message: "a materia with that code already exists",
				})
			}
			if isFKViolation(err) {
				return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
					Error:   "not_found",
					Message: "carrera not found",
				})
			}
			return internalError(c)
		}

		return c.Status(fiber.StatusCreated).JSON(materia)
	}
}

func UpdateMateriaHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid materia id"})
		}

		var req UpdateMateriaRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_request", Message: "invalid body"})
		}

		materia, err := UpdateMateria(c.Context(), db, id, req.Name, req.Semester, req.IsActive)
		if err != nil {
			return internalError(c)
		}
		if materia == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "materia not found"})
		}
		return c.JSON(materia)
	}
}

func DeleteMateriaHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid materia id"})
		}

		hasCursos, err := MateriaHasActiveCursos(c.Context(), db, id)
		if err != nil {
			return internalError(c)
		}
		if hasCursos {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Error:   "conflict",
				Message: "materia has active cursos; deactivate them first",
			})
		}

		f := false
		_, err = UpdateMateria(c.Context(), db, id, nil, nil, &f)
		if err != nil {
			return internalError(c)
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// ─── Student Profiles ─────────────────────────────────────────────────────────

func ListStudentProfilesHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var carreraID *uuid.UUID
		if raw := c.Query("carrera_id"); raw != "" {
			id, err := uuid.Parse(raw)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_param", Message: "invalid carrera_id"})
			}
			carreraID = &id
		}

		var semester *int
		if raw := c.Query("semester"); raw != "" {
			var s int
			if _, err := parseIntQuery(raw, &s); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_param", Message: "invalid semester"})
			}
			semester = &s
		}

		list, err := ListStudentProfiles(c.Context(), db, carreraID, semester)
		if err != nil {
			return internalError(c)
		}
		if list == nil {
			list = []StudentProfile{}
		}
		return c.JSON(list)
	}
}

func CreateStudentProfileHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req CreateStudentProfileRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_request", Message: "invalid body"})
		}
		if req.Email == "" || req.CarreraID == uuid.Nil || req.Semester < 1 || req.Semester > 10 {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "validation_error",
				Message: "email, carrera_id, and semester (1-10) are required",
			})
		}
		if !strings.HasSuffix(req.Email, "@unifranz.edu.bo") {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   "validation_error",
				Message: "email must be a @unifranz.edu.bo address",
			})
		}

		profile, err := CreateStudentProfile(c.Context(), db, req.Email, req.CarreraID, req.Semester, req.StudentCode)
		if err != nil {
			if isUniqueViolation(err) {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
					Error:   "conflict",
					Message: "a student profile with that email already exists",
				})
			}
			return internalError(c)
		}

		return c.Status(fiber.StatusCreated).JSON(profile)
	}
}

func UpdateStudentProfileHandler(db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid profile id"})
		}

		var req UpdateStudentProfileRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_request", Message: "invalid body"})
		}

		profile, err := UpdateStudentProfile(c.Context(), db, id, req)
		if err != nil {
			return internalError(c)
		}
		if profile == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "student profile not found"})
		}
		return c.JSON(profile)
	}
}

