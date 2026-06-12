DROP INDEX IF EXISTS idx_curricular_chunks_embedding;
DROP INDEX IF EXISTS idx_docente_chunks_embedding;

ALTER TABLE rag_curricular_chunks
    ALTER COLUMN embedding TYPE VECTOR(1536);

ALTER TABLE rag_docente_chunks
    ALTER COLUMN embedding TYPE VECTOR(1536);

CREATE INDEX idx_curricular_chunks_embedding
    ON rag_curricular_chunks USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 10);

CREATE INDEX idx_docente_chunks_embedding
    ON rag_docente_chunks USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 10);
