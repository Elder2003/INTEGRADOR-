package rag

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CurricularRepository struct {
	db *pgxpool.Pool
}

func NewCurricularRepository(db *pgxpool.Pool) *CurricularRepository {
	return &CurricularRepository{db: db}
}

func (r *CurricularRepository) InsertDocument(ctx context.Context, doc *CurricularDocument) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO rag_curricular
			(carrera, semestre, codigo_materia, actividad_id, is_rubric,
			 embed_model, embed_dimension, minio_path, original_filename)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id`,
		doc.Carrera, doc.Semestre, doc.CodigoMateria, doc.ActividadID, doc.IsRubric,
		doc.EmbedModel, doc.EmbedDimension, doc.MinioPath, doc.OriginalFilename,
	).Scan(&id)
	return id, err
}

// LockDocument fetches the document row with FOR UPDATE to serialise concurrent ingests.
func (r *CurricularRepository) LockDocument(ctx context.Context, docID uuid.UUID) (*CurricularDocument, error) {
	var d CurricularDocument
	err := r.db.QueryRow(ctx, `
		SELECT id, carrera, semestre, codigo_materia, actividad_id, is_rubric,
		       embed_model, embed_dimension, minio_path, original_filename
		FROM rag_curricular WHERE id=$1 FOR UPDATE`, docID,
	).Scan(&d.ID, &d.Carrera, &d.Semestre, &d.CodigoMateria, &d.ActividadID, &d.IsRubric,
		&d.EmbedModel, &d.EmbedDimension, &d.MinioPath, &d.OriginalFilename)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("curricular document %s not found", docID)
	}
	return &d, err
}

// InsertChunks inserts pending chunks and returns their IDs in order.
func (r *CurricularRepository) InsertChunks(ctx context.Context, docID uuid.UUID, texts []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(texts))
	for i, text := range texts {
		err := r.db.QueryRow(ctx, `
			INSERT INTO rag_curricular_chunks (document_id, chunk_index, chunk_text, status)
			VALUES ($1,$2,$3,'pending') RETURNING id`,
			docID, i, text,
		).Scan(&ids[i])
		if err != nil {
			return nil, fmt.Errorf("insert chunk %d: %w", i, err)
		}
	}
	return ids, nil
}

func (r *CurricularRepository) UpdateChunkEmbedding(ctx context.Context, chunkID uuid.UUID, embedding []float32) error {
	_, err := r.db.Exec(ctx,
		`UPDATE rag_curricular_chunks SET embedding=$1::vector WHERE id=$2`,
		formatVector(embedding), chunkID,
	)
	return err
}

// ActivateChunks atomically deletes old active chunks and promotes pending to active.
func (r *CurricularRepository) ActivateChunks(ctx context.Context, docID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM rag_curricular_chunks WHERE document_id=$1 AND status='active'`, docID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE rag_curricular_chunks SET status='active' WHERE document_id=$1 AND status='pending'`, docID,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *CurricularRepository) DeletePendingChunks(ctx context.Context, docID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM rag_curricular_chunks WHERE document_id=$1 AND status='pending'`, docID,
	)
	return err
}

// GetRubrics returns all active chunks for rubric documents matching the scope + actividad.
func (r *CurricularRepository) GetRubrics(ctx context.Context, carrera string, semestre int, codigoMateria string, actividadID *uuid.UUID) ([]ChunkResult, error) {
	if actividadID == nil {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.document_id, c.chunk_index, c.chunk_text,
		       d.carrera, d.semestre, d.codigo_materia
		FROM rag_curricular_chunks c
		JOIN rag_curricular d ON d.id = c.document_id
		WHERE d.carrera=$1 AND d.semestre=$2 AND d.codigo_materia=$3
		  AND d.actividad_id=$4 AND d.is_rubric=TRUE
		  AND c.status='active'
		ORDER BY c.chunk_index`,
		carrera, semestre, codigoMateria, actividadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCurricularChunks(rows)
}

// QuerySimilar returns the top-N most similar active non-rubric chunks within the given scope.
func (r *CurricularRepository) QuerySimilar(ctx context.Context, carrera string, semestre int, codigoMateria string, queryVec string, limit int) ([]ChunkResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.document_id, c.chunk_index, c.chunk_text,
		       1 - (c.embedding <=> $1::vector) AS similarity_score,
		       d.carrera, d.semestre, d.codigo_materia
		FROM rag_curricular_chunks c
		JOIN rag_curricular d ON d.id = c.document_id
		WHERE d.carrera=$2 AND d.semestre=$3 AND d.codigo_materia=$4
		  AND d.is_rubric=FALSE AND c.status='active'
		  AND c.embedding IS NOT NULL
		ORDER BY c.embedding <=> $1::vector
		LIMIT $5`,
		queryVec, carrera, semestre, codigoMateria, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCurricularChunksWithScore(rows)
}

// QuerySimilarAll returns the top-N most similar active non-rubric chunks across ALL materias.
func (r *CurricularRepository) QuerySimilarAll(ctx context.Context, queryVec string, limit int) ([]ChunkResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.document_id, c.chunk_index, c.chunk_text,
		       1 - (c.embedding <=> $1::vector) AS similarity_score,
		       d.carrera, d.semestre, d.codigo_materia
		FROM rag_curricular_chunks c
		JOIN rag_curricular d ON d.id = c.document_id
		WHERE d.is_rubric=FALSE AND c.status='active'
		  AND c.embedding IS NOT NULL
		ORDER BY c.embedding <=> $1::vector
		LIMIT $2`,
		queryVec, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCurricularChunksWithScore(rows)
}

func (r *CurricularRepository) DeleteDocument(ctx context.Context, id uuid.UUID) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var minioPath string
	err = tx.QueryRow(ctx,
		`SELECT minio_path FROM rag_curricular WHERE id = $1`, id,
	).Scan(&minioPath)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("not_found")
	}
	if err != nil {
		return "", err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM rag_curricular WHERE id = $1`, id); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return minioPath, nil
}

func (r *CurricularRepository) ListDocuments(ctx context.Context, carrera string, semestre int, codigoMateria string, page, limit int) ([]CurricularDocumentSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	whereClause := " WHERE ($1='' OR carrera=$1) AND ($2=0 OR semestre=$2) AND ($3='' OR codigo_materia=$3)"
	args := []any{carrera, semestre, codigoMateria}

	var total int
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM rag_curricular"+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		"SELECT id, carrera, semestre, codigo_materia, original_filename, is_rubric, minio_path, created_at"+
			" FROM rag_curricular"+whereClause+
			" ORDER BY created_at DESC LIMIT $"+strconv.Itoa(len(args)+1)+" OFFSET $"+strconv.Itoa(len(args)+2),
		append(args, limit, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var docs []CurricularDocumentSummary
	for rows.Next() {
		var d CurricularDocumentSummary
		if err := rows.Scan(&d.ID, &d.Carrera, &d.Semestre, &d.CodigoMateria, &d.OriginalFilename, &d.IsRubric, &d.MinioPath, &d.CreatedAt); err != nil {
			return nil, 0, err
		}
		docs = append(docs, d)
	}
	if docs == nil {
		docs = []CurricularDocumentSummary{}
	}
	return docs, total, rows.Err()
}

func (r *CurricularRepository) GetInterdisciplinaryScopes(ctx context.Context, actividadID uuid.UUID) ([]struct {
	Carrera       string
	Semestre      int
	CodigoMateria string
}, error) {
	rows, err := r.db.Query(ctx,
		`SELECT carrera, semestre, codigo_materia
		 FROM actividad_interdisciplinary_scopes
		 WHERE actividad_id = $1`, actividadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scopes []struct {
		Carrera       string
		Semestre      int
		CodigoMateria string
	}
	for rows.Next() {
		var s struct {
			Carrera       string
			Semestre      int
			CodigoMateria string
		}
		if err := rows.Scan(&s.Carrera, &s.Semestre, &s.CodigoMateria); err != nil {
			return nil, err
		}
		scopes = append(scopes, s)
	}
	return scopes, rows.Err()
}

func scanCurricularChunks(rows pgx.Rows) ([]ChunkResult, error) {
	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		var carrera, codigoMateria string
		var semestre int
		if err := rows.Scan(&r.DocumentID, &r.ChunkIndex, &r.ChunkText, &carrera, &semestre, &codigoMateria); err != nil {
			return nil, err
		}
		r.Scope = map[string]any{"carrera": carrera, "semestre": semestre, "codigo_materia": codigoMateria}
		results = append(results, r)
	}
	return results, rows.Err()
}

func scanCurricularChunksWithScore(rows pgx.Rows) ([]ChunkResult, error) {
	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		var carrera, codigoMateria string
		var semestre int
		if err := rows.Scan(&r.DocumentID, &r.ChunkIndex, &r.ChunkText, &r.SimilarityScore, &carrera, &semestre, &codigoMateria); err != nil {
			return nil, err
		}
		r.Scope = map[string]any{"carrera": carrera, "semestre": semestre, "codigo_materia": codigoMateria}
		results = append(results, r)
	}
	return results, rows.Err()
}
