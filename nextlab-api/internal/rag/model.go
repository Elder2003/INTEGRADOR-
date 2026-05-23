package rag

import (
	"time"

	"github.com/google/uuid"
)

type CurricularDocumentSummary struct {
	ID               uuid.UUID `json:"id"`
	Carrera          string    `json:"carrera"`
	Semestre         int       `json:"semestre"`
	CodigoMateria    string    `json:"codigo_materia"`
	OriginalFilename string    `json:"original_filename"`
	IsRubric         bool      `json:"is_rubric"`
	MinioPath        string    `json:"minio_path"`
	CreatedAt        time.Time `json:"created_at"`
}

type CurricularDocument struct {
	ID               uuid.UUID
	Carrera          string
	Semestre         int
	CodigoMateria    string
	ActividadID      *uuid.UUID
	IsRubric         bool
	EmbedModel       string
	EmbedDimension   int
	MinioPath        string
	OriginalFilename string
}

type DocenteDocument struct {
	ID               uuid.UUID
	DocenteID        uuid.UUID
	CursoID          uuid.UUID
	ActividadID      *uuid.UUID
	IsRubric         bool
	EmbedModel       string
	EmbedDimension   int
	MinioPath        string
	OriginalFilename string
}

type ChunkResult struct {
	DocumentID      uuid.UUID
	ChunkIndex      int
	ChunkText       string
	SimilarityScore float32
	Scope           map[string]any
}

type IngestResult struct {
	DocumentID uuid.UUID `json:"document_id"`
	ChunkCount int       `json:"chunk_count"`
	Status     string    `json:"status"`
}

type RetrievalResponse struct {
	Rubrics []ChunkResult `json:"rubrics"`
	Chunks  []ChunkResult `json:"chunks"`
}

type IngestCurricularRequest struct {
	FileData         []byte
	FileSize         int64
	OriginalFilename string
	Carrera          string
	Semestre         int
	CodigoMateria    string
	IsRubric         bool
	DocumentID       *uuid.UUID
	ActividadID      *uuid.UUID
	ExternalURL      string
}

type IngestDocenteRequest struct {
	FileData         []byte
	FileSize         int64
	OriginalFilename string
	DocenteID        uuid.UUID
	CursoID          uuid.UUID
	ActividadID      *uuid.UUID
	IsRubric         bool
	DocumentID       *uuid.UUID
	ExternalURL      string
}

type CurricularQueryRequest struct {
	Carrera       string
	Semestre      int
	CodigoMateria string
	Query         string
	MaxChunks     int
	ActividadID   *uuid.UUID
}

type DocenteQueryRequest struct {
	DocenteID   uuid.UUID
	CursoID     uuid.UUID
	ActividadID *uuid.UUID
	Query       string
	MaxChunks   int
}
