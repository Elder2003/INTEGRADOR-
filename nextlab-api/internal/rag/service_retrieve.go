package rag

import (
	"context"
	"fmt"

	"github.com/Elder2003/nextlab-api/internal/config"
)

type RetrieveService struct {
	curRepo     *CurricularRepository
	docRepo     *DocenteRepository
	embedClient *EmbedClient
	cfg         *config.Config
}

func NewRetrieveService(
	curRepo *CurricularRepository,
	docRepo *DocenteRepository,
	embedClient *EmbedClient,
	cfg *config.Config,
) *RetrieveService {
	return &RetrieveService{
		curRepo:     curRepo,
		docRepo:     docRepo,
		embedClient: embedClient,
		cfg:         cfg,
	}
}

// QueryCurricular embeds the query, fetches rubrics (full, no similarity ranking),
// then runs a scoped similarity search on non-rubric active chunks.
// If the actividad has interdisciplinary scopes, also queries each extra scope
// and returns a deduplicated union of all chunks.
// Rubrics excluded from token budget per FR-005/SC-004.
func (s *RetrieveService) QueryCurricular(ctx context.Context, req CurricularQueryRequest) (*RetrievalResponse, error) {
	queryVec, err := s.embedClient.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// Rubrics fetched first — returned in full, not ranked by similarity
	rubrics, err := s.curRepo.GetRubrics(ctx, req.Carrera, req.Semestre, req.CodigoMateria, req.ActividadID)
	if err != nil {
		return nil, fmt.Errorf("get rubrics: %w", err)
	}

	limit := s.resolveLimit(req.MaxChunks)
	vecStr := formatVector(queryVec)

	chunks, err := s.curRepo.QuerySimilar(ctx, req.Carrera, req.Semestre, req.CodigoMateria, vecStr, limit)
	if err != nil {
		return nil, fmt.Errorf("similarity query: %w", err)
	}

	// Union extra interdisciplinary scopes when actividadID is set
	if req.ActividadID != nil {
		extraScopes, err := s.curRepo.GetInterdisciplinaryScopes(ctx, *req.ActividadID)
		if err != nil {
			return nil, fmt.Errorf("get interdisciplinary scopes: %w", err)
		}
		for _, scope := range extraScopes {
			extra, err := s.curRepo.QuerySimilar(ctx, scope.Carrera, scope.Semestre, scope.CodigoMateria, vecStr, limit)
			if err != nil {
				return nil, fmt.Errorf("similarity query (extra scope): %w", err)
			}
			chunks = append(chunks, extra...)
		}
		// Deduplicate by DocumentID, keeping first occurrence
		seen := make(map[string]bool, len(chunks))
		deduped := chunks[:0]
		for _, ch := range chunks {
			key := ch.DocumentID.String() + ":" + fmt.Sprintf("%d", ch.ChunkIndex)
			if !seen[key] {
				seen[key] = true
				deduped = append(deduped, ch)
			}
		}
		chunks = deduped
	}

	return &RetrievalResponse{
		Rubrics: nullSafe(rubrics),
		Chunks:  nullSafe(chunks),
	}, nil
}

// QueryDocente embeds the query, fetches rubrics, then runs a scoped similarity
// search on non-rubric active chunks for the given docente's course.
// Rubrics excluded from token budget per FR-005/SC-004.
func (s *RetrieveService) QueryDocente(ctx context.Context, req DocenteQueryRequest) (*RetrievalResponse, error) {
	queryVec, err := s.embedClient.Embed(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// Rubrics fetched first — returned in full, not ranked by similarity
	rubrics, err := s.docRepo.GetRubrics(ctx, req.DocenteID, req.CursoID, req.ActividadID)
	if err != nil {
		return nil, fmt.Errorf("get rubrics: %w", err)
	}

	limit := s.resolveLimit(req.MaxChunks)
	chunks, err := s.docRepo.QuerySimilar(ctx, req.DocenteID, req.CursoID, req.ActividadID, formatVector(queryVec), limit)
	if err != nil {
		return nil, fmt.Errorf("similarity query: %w", err)
	}

	return &RetrievalResponse{
		Rubrics: nullSafe(rubrics),
		Chunks:  nullSafe(chunks),
	}, nil
}

func (s *RetrieveService) resolveLimit(requested int) int {
	if requested > 0 {
		return requested
	}
	// Default: derive from token budget / approximate chunk token size
	approxTokensPerChunk := s.cfg.RAGChunkTokens
	if approxTokensPerChunk <= 0 {
		approxTokensPerChunk = 512
	}
	n := s.cfg.RAGTokenBudget / approxTokensPerChunk
	if n <= 0 {
		n = 4
	}
	return n
}

// nullSafe returns an empty slice instead of nil to produce [] in JSON.
func nullSafe(s []ChunkResult) []ChunkResult {
	if s == nil {
		return []ChunkResult{}
	}
	return s
}
