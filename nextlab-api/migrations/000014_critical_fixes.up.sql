-- M14: Fix embedding dimension (nomic-embed-text = 768 dims, not 1536)
-- NOTE: Existing embeddings stored with wrong dimension are invalidated.
-- Re-ingest all rag_curricular and rag_docente documents after applying.

DROP INDEX IF EXISTS idx_curricular_chunks_embedding;
DROP INDEX IF EXISTS idx_docente_chunks_embedding;

ALTER TABLE rag_curricular_chunks
    ALTER COLUMN embedding TYPE VECTOR(768);

ALTER TABLE rag_docente_chunks
    ALTER COLUMN embedding TYPE VECTOR(768);

CREATE INDEX idx_curricular_chunks_embedding
    ON rag_curricular_chunks USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 10);

CREATE INDEX idx_docente_chunks_embedding
    ON rag_docente_chunks USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 10);
