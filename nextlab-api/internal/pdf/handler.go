package pdf

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func ActivityReportHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		actividadID, err := uuid.Parse(c.Params("aid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_actividad_id"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), pdfTimeout)
		defer cancel()

		pdfBytes, err := svc.GenerateActivityReport(ctx, actividadID)
		if err != nil {
			if ctx.Err() != nil {
				return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{"error": "pdf_timeout"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		filename := fmt.Sprintf("actividad-%s-reporte.pdf", actividadID.String()[:8])
		return sendPDF(c, pdfBytes, filename)
	}
}

func DetectionReportHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		submissionID, err := uuid.Parse(c.Params("subid"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_submission_id"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), pdfTimeout)
		defer cancel()

		pdfBytes, err := svc.GenerateDetectionReport(ctx, submissionID)
		if err != nil {
			if ctx.Err() != nil {
				return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{"error": "pdf_timeout"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		filename := fmt.Sprintf("detection-%s.pdf", submissionID.String()[:8])
		return sendPDF(c, pdfBytes, filename)
	}
}

func AppealAuditReportHandler(svc *Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		appealID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_appeal_id"})
		}

		ctx, cancel := context.WithTimeout(c.Context(), pdfTimeout)
		defer cancel()

		pdfBytes, err := svc.GenerateAppealAuditReport(ctx, appealID)
		if err != nil {
			if ctx.Err() != nil {
				return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{"error": "pdf_timeout"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal_error"})
		}

		filename := fmt.Sprintf("appeal-%s-audit.pdf", appealID.String()[:8])
		return sendPDF(c, pdfBytes, filename)
	}
}

func sendPDF(c *fiber.Ctx, pdfBytes []byte, filename string) error {
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Send(pdfBytes)
}
