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
	chunks, err := s.curRepo.QuerySimilar(ctx, req.Carrera, req.Semestre, req.CodigoMateria, formatVector(queryVec), limit)
	if err != nil {
		return nil, fmt.Errorf("similarity query: %w", err)
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
