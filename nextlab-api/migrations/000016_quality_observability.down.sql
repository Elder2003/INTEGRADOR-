DROP TRIGGER IF EXISTS trg_docente_search_vector    ON rag_docente_chunks;
DROP TRIGGER IF EXISTS trg_curricular_search_vector ON rag_curricular_chunks;
DROP FUNCTION IF EXISTS update_rag_search_vector();
DROP INDEX IF EXISTS idx_docente_chunks_fts;
DROP INDEX IF EXISTS idx_curricular_chunks_fts;
ALTER TABLE rag_docente_chunks    DROP COLUMN IF EXISTS search_vector;
ALTER TABLE rag_curricular_chunks DROP COLUMN IF EXISTS search_vector;
ALTER TABLE analysis_results DROP COLUMN IF EXISTS temporal_metrics;
DROP INDEX IF EXISTS uq_appeal_session_type_active;
-- Note: cannot safely restore UNIQUE(session_id) if duplicate (session_id,type) data now exists
