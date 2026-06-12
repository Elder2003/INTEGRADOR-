package rag

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DocenteRepository struct {
	db *pgxpool.Pool
}

func NewDocenteRepository(db *pgxpool.Pool) *DocenteRepository {
	return &DocenteRepository{db: db}
}

func (r *DocenteRepository) InsertDocument(ctx context.Context, doc *DocenteDocument) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO rag_docente
			(docente_id, curso_id, actividad_id, is_rubric,
			 embed_model, embed_dimension, minio_path, original_filename)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id`,
		doc.DocenteID, doc.CursoID, doc.ActividadID, doc.IsRubric,
		doc.EmbedModel, doc.EmbedDimension, doc.MinioPath, doc.OriginalFilename,
	).Scan(&id)
	return id, err
}

// LockDocument fetches the document row with FOR UPDATE to serialise concurrent ingests.
func (r *DocenteRepository) LockDocument(ctx context.Context, docID uuid.UUID) (*DocenteDocument, error) {
	var d DocenteDocument
	err := r.db.QueryRow(ctx, `
		SELECT id, docente_id, curso_id, actividad_id, is_rubric,
		       embed_model, embed_dimension, minio_path, original_filename
		FROM rag_docente WHERE id=$1 FOR UPDATE`, docID,
	).Scan(&d.ID, &d.DocenteID, &d.CursoID, &d.ActividadID, &d.IsRubric,
		&d.EmbedModel, &d.EmbedDimension, &d.MinioPath, &d.OriginalFilename)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("docente document %s not found", docID)
	}
	return &d, err
}

// InsertChunks inserts pending chunks and returns their IDs in order.
func (r *DocenteRepository) InsertChunks(ctx context.Context, docID uuid.UUID, texts []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, len(texts))
	for i, text := range texts {
		err := r.db.QueryRow(ctx, `
			INSERT INTO rag_docente_chunks (document_id, chunk_index, chunk_text, status)
			VALUES ($1,$2,$3,'pending') RETURNING id`,
			docID, i, text,
		).Scan(&ids[i])
		if err != nil {
			return nil, fmt.Errorf("insert chunk %d: %w", i, err)
		}
	}
	return ids, nil
}

func (r *DocenteRepository) UpdateChunkEmbedding(ctx context.Context, chunkID uuid.UUID, embedding []float32) error {
	_, err := r.db.Exec(ctx,
		`UPDATE rag_docente_chunks SET embedding=$1::vector WHERE id=$2`,
		formatVector(embedding), chunkID,
	)
	return err
}

// ActivateChunks atomically deletes old active chunks and promotes pending to active.
func (r *DocenteRepository) ActivateChunks(ctx context.Context, docID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`DELETE FROM rag_docente_chunks WHERE document_id=$1 AND status='active'`, docID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE rag_docente_chunks SET status='active' WHERE document_id=$1 AND status='pending'`, docID,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *DocenteRepository) DeletePendingChunks(ctx context.Context, docID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM rag_docente_chunks WHERE document_id=$1 AND status='pending'`, docID,
	)
	return err
}

// GetRubrics returns all active chunks for rubric documents matching the scope + actividad.
func (r *DocenteRepository) GetRubrics(ctx context.Context, docenteID, cursoID uuid.UUID, actividadID *uuid.UUID) ([]ChunkResult, error) {
	if actividadID == nil {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT c.document_id, c.chunk_index, c.chunk_text,
		       d.docente_id, d.curso_id
		FROM rag_docente_chunks c
		JOIN rag_docente d ON d.id = c.document_id
		WHERE d.docente_id=$1 AND d.curso_id=$2
		  AND d.actividad_id=$3 AND d.is_rubric=TRUE
		  AND c.status='active'
		ORDER BY c.chunk_index`,
		docenteID, cursoID, actividadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocenteChunks(rows)
}

// QuerySimilar returns the top-N most similar active non-rubric chunks within the docente's scope.
// actividadID is an optional additional filter.
func (r *DocenteRepository) QuerySimilar(ctx context.Context, docenteID, cursoID uuid.UUID, actividadID *uuid.UUID, queryVec string, limit int) ([]ChunkResult, error) {
	var rows pgx.Rows
	var err error

	if actividadID != nil {
		rows, err = r.db.Query(ctx, `
			SELECT c.document_id, c.chunk_index, c.chunk_text,
			       1 - (c.embedding <=> $1::vector) AS similarity_score,
			       d.docente_id, d.curso_id
			FROM rag_docente_chunks c
			JOIN rag_docente d ON d.id = c.document_id
			WHERE d.docente_id=$2 AND d.curso_id=$3
			  AND d.actividad_id=$4
			  AND d.is_rubric=FALSE AND c.status='active'
			  AND c.embedding IS NOT NULL
			ORDER BY c.embedding <=> $1::vector
			LIMIT $5`,
			queryVec, docenteID, cursoID, actividadID, limit,
		)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT c.document_id, c.chunk_index, c.chunk_text,
			       1 - (c.embedding <=> $1::vector) AS similarity_score,
			       d.docente_id, d.curso_id
			FROM rag_docente_chunks c
			JOIN rag_docente d ON d.id = c.document_id
			WHERE d.docente_id=$2 AND d.curso_id=$3
			  AND d.is_rubric=FALSE AND c.status='active'
			  AND c.embedding IS NOT NULL
			ORDER BY c.embedding <=> $1::vector
			LIMIT $4`,
			queryVec, docenteID, cursoID, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocenteChunksWithScore(rows)
}

// LexicalSearchDocente performs full-text BM25-style search using tsvector for docente chunks.
func (r *DocenteRepository) LexicalSearchDocente(ctx context.Context, docenteID, cursoID uuid.UUID, query string, limit int) ([]ChunkResult, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.document_id, c.chunk_index, c.chunk_text,
		       ts_rank(c.search_vector, plainto_tsquery('pg_catalog.spanish', $3))::float4 AS score,
		       d.docente_id, d.curso_id
		FROM rag_docente_chunks c
		JOIN rag_docente d ON d.id = c.document_id
		WHERE d.docente_id=$1 AND d.curso_id=$2
		  AND d.is_rubric=FALSE AND c.status='active'
		  AND c.search_vector @@ plainto_tsquery('pg_catalog.spanish', $3)
		ORDER BY score DESC
		LIMIT $4`,
		docenteID, cursoID, query, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocenteChunksWithScore(rows)
}

// VerifyDocenteOwnsCourse returns true if the given docente owns (or teaches) the given curso.
func (r *DocenteRepository) VerifyDocenteOwnsCourse(ctx context.Context, docenteID, cursoID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(1) FROM cursos WHERE id=$1 AND docente_id=$2`,
		cursoID, docenteID,
	).Scan(&count)
	return count > 0, err
}

func scanDocenteChunks(rows pgx.Rows) ([]ChunkResult, error) {
	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		var docenteID, cursoID uuid.UUID
		if err := rows.Scan(&r.DocumentID, &r.ChunkIndex, &r.ChunkText, &docenteID, &cursoID); err != nil {
			return nil, err
		}
		r.Scope = map[string]any{"docente_id": docenteID, "curso_id": cursoID}
		results = append(results, r)
	}
	return results, rows.Err()
}

func scanDocenteChunksWithScore(rows pgx.Rows) ([]ChunkResult, error) {
	var results []ChunkResult
	for rows.Next() {
		var r ChunkResult
		var docenteID, cursoID uuid.UUID
		if err := rows.Scan(&r.DocumentID, &r.ChunkIndex, &r.ChunkText, &r.SimilarityScore, &docenteID, &cursoID); err != nil {
			return nil, err
		}
		r.Scope = map[string]any{"docente_id": docenteID, "curso_id": cursoID}
		results = append(results, r)
	}
	return results, rows.Err()
}
