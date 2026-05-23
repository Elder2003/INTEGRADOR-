package admin

import (
	"crypto/rand"
	"log/slog"
	"math/big"
	"regexp"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var docenteEmailRe = regexp.MustCompile(`^doc\..+@unifranz\.edu\.bo$`)

const passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generatePassword(length int) (string, error) {
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
		if err != nil {
			return "", err
		}
		b[i] = passwordChars[n.Int64()]
	}
	return string(b), nil
}

func CreateDocenteHandler(repo *Repository, appEnv string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req CreateDocenteRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "invalid request body"})
		}
		if !docenteEmailRe.MatchString(req.Email) {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "email must match ^doc\\..+@unifranz\\.edu\\.bo$"})
		}
		if req.FullName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "full_name is required"})
		}

		var plainPassword string
		var passwordHash string

		if appEnv == "development" && req.TestPassword != nil && *req.TestPassword != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(*req.TestPassword), bcrypt.DefaultCost)
			if err != nil {
				slog.ErrorContext(c.Context(), "bcrypt error", "err", err)
				return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
			}
			passwordHash = string(hash)
			plainPassword = *req.TestPassword
		} else {
			plain, err := generatePassword(12)
			if err != nil {
				slog.ErrorContext(c.Context(), "password generation error", "err", err)
				return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
			if err != nil {
				slog.ErrorContext(c.Context(), "bcrypt error", "err", err)
				return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
			}
			passwordHash = string(hash)
			plainPassword = plain
		}

		docente, err := repo.InsertDocente(c.Context(), req.Email, req.FullName, passwordHash)
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Error: "conflict", Message: "email already registered"})
			}
			slog.ErrorContext(c.Context(), "insert docente error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":                 docente.ID,
			"email":              docente.Email,
			"full_name":          docente.FullName,
			"is_active":          docente.IsActive,
			"created_at":         docente.CreatedAt,
			"updated_at":         docente.UpdatedAt,
			"temporary_password": plainPassword,
		})
	}
}

func ListDocentesHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, _ := strconv.Atoi(c.Query("page", "1"))
		limit, _ := strconv.Atoi(c.Query("limit", "20"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}
		offset := (page - 1) * limit

		var activeOnly *bool
		if active := c.Query("active"); active != "" {
			v, err := strconv.ParseBool(active)
			if err == nil {
				activeOnly = &v
			}
		}

		docentes, total, err := repo.ListDocentes(c.Context(), activeOnly, limit, offset)
		if err != nil {
			slog.ErrorContext(c.Context(), "list docentes error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}

		return c.JSON(fiber.Map{
			"docentes": docentes,
			"total":    total,
			"page":     page,
			"limit":    limit,
		})
	}
}

func GetDocenteHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid docente id"})
		}
		docente, err := repo.GetDocente(c.Context(), id)
		if err != nil {
			slog.ErrorContext(c.Context(), "get docente error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if docente == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "docente not found"})
		}
		return c.JSON(docente)
	}
}

func UpdateDocenteHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid docente id"})
		}
		var req UpdateDocenteRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "validation_error", Message: "invalid request body"})
		}
		docente, err := repo.UpdateDocente(c.Context(), id, req.FullName, req.IsActive)
		if err != nil {
			slog.ErrorContext(c.Context(), "update docente error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if docente == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "docente not found"})
		}
		return c.JSON(docente)
	}
}

func DeleteDocenteHandler(repo *Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid_id", Message: "invalid docente id"})
		}
		f := false
		docente, err := repo.UpdateDocente(c.Context(), id, nil, &f)
		if err != nil {
			slog.ErrorContext(c.Context(), "delete docente error", "err", err)
			return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: "internal_error", Message: "unexpected error"})
		}
		if docente == nil {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "not_found", Message: "docente not found"})
		}
		return c.SendStatus(fiber.StatusNoContent)
	}
}
