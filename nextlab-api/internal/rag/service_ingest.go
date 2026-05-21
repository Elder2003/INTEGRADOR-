package rag

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"

	"github.com/Elder2003/nextlab-api/internal/config"
)

const ragBucket = "rag-docs"

type IngestService struct {
	curRepo     *CurricularRepository
	docRepo     *DocenteRepository
	embedClient *EmbedClient
	minio       *minio.Client
	cfg         *config.Config
}

func NewIngestService(
	curRepo *CurricularRepository,
	docRepo *DocenteRepository,
	embedClient *EmbedClient,
	minioClient *minio.Client,
	cfg *config.Config,
) *IngestService {
	return &IngestService{
		curRepo:     curRepo,
		docRepo:     docRepo,
		embedClient: embedClient,
		minio:       minioClient,
		cfg:         cfg,
	}
}

func (s *IngestService) IngestCurricular(ctx context.Context, req IngestCurricularRequest) (*IngestResult, error) {
	ext := strings.ToLower(filepath.Ext(req.OriginalFilename))
	text, err := ExtractText(req.FileData, ext)
	if err != nil {
		return nil, err // "unsupported file type" passes through as-is
	}
	if req.ExternalURL != "" {
		fetched, fErr := FetchURL(ctx, req.ExternalURL)
		if fErr != nil {
			slog.Warn("external URL fetch failed", "url", req.ExternalURL, "err", fErr)
		} else {
			text = text + "\n" + fetched
		}
	}

	// Rubric documents stored as a single unsplit unit per FR-005 and Constitution Principle V
	var chunks []string
	if req.IsRubric {
		chunks = []string{text}
	} else {
		chunks = ChunkText(text, s.cfg.RAGChunkTokens, s.cfg.RAGChunkOverlap)
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no text could be extracted from the document")
	}

	isReIngest := req.DocumentID != nil
	var docID uuid.UUID
	if isReIngest {
		docID = *req.DocumentID
	} else {
		docID = uuid.New()
	}

	minioPath := fmt.Sprintf("curricular/%s/original%s", docID, ext)
	if _, err := s.minio.PutObject(ctx, ragBucket, minioPath,
		bytes.NewReader(req.FileData), int64(len(req.FileData)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	); err != nil {
		return nil, fmt.Errorf("upload to minio: %w", err)
	}

	if !isReIngest {
		doc := &CurricularDocument{
			Carrera:          req.Carrera,
			Semestre:         req.Semestre,
			CodigoMateria:    req.CodigoMateria,
			ActividadID:      req.ActividadID,
			IsRubric:         req.IsRubric,
			EmbedModel:       s.cfg.AIEmbedModel,
			EmbedDimension:   s.cfg.AIEmbedDimension,
			MinioPath:        minioPath,
			OriginalFilename: req.OriginalFilename,
		}
		newID, err := s.curRepo.InsertDocument(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("insert curricular document: %w", err)
		}
		docID = newID
	}

	if err := s.ingestCurricularChunks(ctx, docID, chunks); err != nil {
		return nil, err
	}

	status := "ingested"
	if isReIngest {
		status = "re-ingested"
	}
	return &IngestResult{DocumentID: docID, ChunkCount: len(chunks), Status: status}, nil
}

func (s *IngestService) IngestDocente(ctx context.Context, req IngestDocenteRequest) (*IngestResult, error) {
	ext := strings.ToLower(filepath.Ext(req.OriginalFilename))
	text, err := ExtractText(req.FileData, ext)
	if err != nil {
		return nil, err
	}
	if req.ExternalURL != "" {
		fetched, fErr := FetchURL(ctx, req.ExternalURL)
		if fErr != nil {
			slog.Warn("external URL fetch failed", "url", req.ExternalURL, "err", fErr)
		} else {
			text = text + "\n" + fetched
		}
	}

	// Rubric documents stored as a single unsplit unit per FR-005 and Constitution Principle V
	var chunks []string
	if req.IsRubric {
		chunks = []string{text}
	} else {
		chunks = ChunkText(text, s.cfg.RAGChunkTokens, s.cfg.RAGChunkOverlap)
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no text could be extracted from the document")
	}

	isReIngest := req.DocumentID != nil
	var docID uuid.UUID
	if isReIngest {
		docID = *req.DocumentID
	} else {
		docID = uuid.New()
	}

	minioPath := fmt.Sprintf("docente/%s/original%s", docID, ext)
	if _, err := s.minio.PutObject(ctx, ragBucket, minioPath,
		bytes.NewReader(req.FileData), int64(len(req.FileData)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	); err != nil {
		return nil, fmt.Errorf("upload to minio: %w", err)
	}

	if !isReIngest {
		doc := &DocenteDocument{
			DocenteID:        req.DocenteID,
			CursoID:          req.CursoID,
			ActividadID:      req.ActividadID,
			IsRubric:         req.IsRubric,
			EmbedModel:       s.cfg.AIEmbedModel,
			EmbedDimension:   s.cfg.AIEmbedDimension,
			MinioPath:        minioPath,
			OriginalFilename: req.OriginalFilename,
		}
		newID, err := s.docRepo.InsertDocument(ctx, doc)
		if err != nil {
			return nil, fmt.Errorf("insert docente document: %w", err)
		}
		docID = newID
	}

	if err := s.ingestDocenteChunks(ctx, docID, chunks); err != nil {
		return nil, err
	}

	status := "ingested"
	if isReIngest {
		status = "re-ingested"
	}
	return &IngestResult{DocumentID: docID, ChunkCount: len(chunks), Status: status}, nil
}

// ingestCurricularChunks runs the two-phase pending→active swap for curricular chunks.
// On any embedding failure, pending chunks are deleted and old active chunks survive intact.
func (s *IngestService) ingestCurricularChunks(ctx context.Context, docID uuid.UUID, texts []string) error {
	chunkIDs, err := s.curRepo.InsertChunks(ctx, docID, texts)
	if err != nil {
		return fmt.Errorf("insert pending chunks: %w", err)
	}

	for i, text := range texts {
		vec, err := s.embedClient.Embed(ctx, text)
		if err != nil {
			slog.Error("embedding failed; rolling back pending chunks", "doc_id", docID, "chunk", i, "err", err)
			_ = s.curRepo.DeletePendingChunks(ctx, docID)
			return fmt.Errorf("embedding_failed")
		}
		if err := s.curRepo.UpdateChunkEmbedding(ctx, chunkIDs[i], vec); err != nil {
			_ = s.curRepo.DeletePendingChunks(ctx, docID)
			return fmt.Errorf("embedding_failed")
		}
	}

	if err := s.curRepo.ActivateChunks(ctx, docID); err != nil {
		return fmt.Errorf("activate chunks: %w", err)
	}
	return nil
}

// ingestDocenteChunks runs the two-phase pending→active swap for docente chunks.
func (s *IngestService) ingestDocenteChunks(ctx context.Context, docID uuid.UUID, texts []string) error {
	chunkIDs, err := s.docRepo.InsertChunks(ctx, docID, texts)
	if err != nil {
		return fmt.Errorf("insert pending chunks: %w", err)
	}

	for i, text := range texts {
		vec, err := s.embedClient.Embed(ctx, text)
		if err != nil {
			slog.Error("embedding failed; rolling back pending chunks", "doc_id", docID, "chunk", i, "err", err)
			_ = s.docRepo.DeletePendingChunks(ctx, docID)
			return fmt.Errorf("embedding_failed")
		}
		if err := s.docRepo.UpdateChunkEmbedding(ctx, chunkIDs[i], vec); err != nil {
			_ = s.docRepo.DeletePendingChunks(ctx, docID)
			return fmt.Errorf("embedding_failed")
		}
	}

	if err := s.docRepo.ActivateChunks(ctx, docID); err != nil {
		return fmt.Errorf("activate chunks: %w", err)
	}
	return nil
}
