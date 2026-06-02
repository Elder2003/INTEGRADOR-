-- M16: Quality & Observability improvements

-- Fix 2: Appeals — replace simple UNIQUE(session_id) with partial index per type+status
ALTER TABLE appeals DROP CONSTRAINT IF EXISTS appeals_session_id_key;
CREATE UNIQUE INDEX uq_appeal_session_type_active
    ON appeals (session_id, type)
    WHERE status IN ('pending', 'escalated');

-- Fix 3: Temporal heuristic metrics in analysis_results
ALTER TABLE analysis_results ADD COLUMN IF NOT EXISTS temporal_metrics JSONB;

-- Fix 4: Hybrid RAG — full-text search vectors
ALTER TABLE rag_curricular_chunks ADD COLUMN IF NOT EXISTS search_vector TSVECTOR;
ALTER TABLE rag_docente_chunks    ADD COLUMN IF NOT EXISTS search_vector TSVECTOR;

CREATE INDEX IF NOT EXISTS idx_curricular_chunks_fts ON rag_curricular_chunks USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_docente_chunks_fts    ON rag_docente_chunks    USING GIN(search_vector);

CREATE OR REPLACE FUNCTION update_rag_search_vector()
RETURNS TRIGGER AS $$
BEGIN
  NEW.search_vector := to_tsvector('pg_catalog.spanish', coalesce(NEW.content, ''));
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_curricular_search_vector
BEFORE INSERT OR UPDATE ON rag_curricular_chunks
FOR EACH ROW EXECUTE FUNCTION update_rag_search_vector();

CREATE TRIGGER trg_docente_search_vector
BEFORE INSERT OR UPDATE ON rag_docente_chunks
FOR EACH ROW EXECUTE FUNCTION update_rag_search_vector();

-- Backfill existing chunks
UPDATE rag_curricular_chunks SET search_vector = to_tsvector('pg_catalog.spanish', coalesce(content, ''));
UPDATE rag_docente_chunks    SET search_vector = to_tsvector('pg_catalog.spanish', coalesce(content, ''));
