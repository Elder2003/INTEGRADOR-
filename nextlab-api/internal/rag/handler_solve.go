package rag

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/Elder2003/nextlab-api/internal/ai"
)

func solveProblemHandler(curRepo *CurricularRepository, embedClient *EmbedClient, aiSvc *ai.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req SolveProblemRequest
		if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Problem) == "" {
			return c.Status(400).JSON(fiber.Map{
				"error":   "invalid_request",
				"message": "problem is required",
			})
		}

		// Embed the problem description
		queryVec, err := embedClient.Embed(c.Context(), req.Problem)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "embed_error", "message": "failed to process query"})
		}

		// Retrieve top chunks across ALL materias (wider limit so we cover more materias)
		chunks, err := curRepo.QuerySimilarAll(c.Context(), formatVector(queryVec), 30)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "internal_error", "message": "retrieval failed"})
		}

		// Group by materia — keep only the best chunk per materia
		type materiaKey struct {
			codigo  string
			carrera string
			sem     int
		}
		best := make(map[materiaKey]ChunkResult)
		for _, ch := range chunks {
			k := materiaKey{
				codigo:  fmt.Sprintf("%v", ch.Scope["codigo_materia"]),
				carrera: fmt.Sprintf("%v", ch.Scope["carrera"]),
				sem:     int(ch.Scope["semestre"].(int)),
			}
			if existing, ok := best[k]; !ok || ch.SimilarityScore > existing.SimilarityScore {
				best[k] = ch
			}
		}

		// Build sorted materia suggestions (filter out very low similarity)
		suggestions := make([]MateriaSuggestion, 0, len(best))
		for k, ch := range best {
			if ch.SimilarityScore < 0.2 {
				continue
			}
			excerpt := ch.ChunkText
			if len(excerpt) > 300 {
				excerpt = excerpt[:300] + "…"
			}
			suggestions = append(suggestions, MateriaSuggestion{
				CodigoMateria:  k.codigo,
				Carrera:        k.carrera,
				Semestre:       k.sem,
				RelevanceScore: ch.SimilarityScore,
				Excerpt:        excerpt,
			})
		}

		// Sort by relevance descending
		for i := 1; i < len(suggestions); i++ {
			for j := i; j > 0 && suggestions[j].RelevanceScore > suggestions[j-1].RelevanceScore; j-- {
				suggestions[j], suggestions[j-1] = suggestions[j-1], suggestions[j]
			}
		}

		// Build AI prompt from top matches
		suggestion := ""
		if aiSvc != nil && len(suggestions) > 0 {
			top := suggestions
			if len(top) > 5 {
				top = top[:5]
			}
			var contextParts []string
			for _, s := range top {
				contextParts = append(contextParts,
					fmt.Sprintf("Materia %s (Carrera: %s, Semestre: %d, Relevancia: %.0f%%):\n%s",
						s.CodigoMateria, s.Carrera, s.Semestre, s.RelevanceScore*100, s.Excerpt),
				)
			}
			context := strings.Join(contextParts, "\n\n---\n\n")

			prompt := fmt.Sprintf(
				"Eres un asesor académico. Un administrador ha planteado la siguiente problemática:\n\n"+
					"\"%s\"\n\n"+
					"Se encontró el siguiente contenido curricular relevante:\n\n%s\n\n"+
					"Basándote en este contenido, redacta en español:\n"+
					"1. Un análisis breve de cómo estas materias abordan la problemática.\n"+
					"2. Cuáles materias son más relevantes y por qué.\n"+
					"3. Una recomendación práctica para abordar el problema usando estos recursos académicos.\n\n"+
					"Sé conciso y directo. Máximo 250 palabras.",
				req.Problem, context,
			)

			dispatchReq := ai.DispatchRequest{
				RequestID: uuid.New(),
				Messages: []ai.ChatMessage{
					{Role: "user", Content: prompt},
				},
				MaxTokens: 512,
			}

			resp, _, aiErr := aiSvc.Dispatch(c.Context(), dispatchReq, 30)
			if aiErr == nil && len(resp.Choices) > 0 {
				suggestion = resp.Choices[0].Message.Content
			}
		}

		if suggestion == "" && len(suggestions) > 0 {
			// Fallback when AI is unavailable
			names := make([]string, 0, len(suggestions))
			for _, s := range suggestions {
				if len(names) >= 3 {
					break
				}
				names = append(names, s.CodigoMateria)
			}
			suggestion = fmt.Sprintf(
				"Se encontraron %d materia(s) relevantes para esta problemática. Las más relacionadas son: %s. "+
					"Revisa los contenidos destacados de cada materia para más detalle.",
				len(suggestions), strings.Join(names, ", "),
			)
		}

		return c.JSON(SolveProblemResponse{
			Problem:    req.Problem,
			Suggestion: suggestion,
			Materias:   suggestions,
		})
	}
}
