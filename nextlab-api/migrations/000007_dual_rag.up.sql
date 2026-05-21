-- M7: Dual RAG Pipeline

-- Curricular knowledge base documents
CREATE TABLE rag_curricular (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    carrera          TEXT        NOT NULL,
    semestre         SMALLINT    NOT NULL CHECK (semestre BETWEEN 1 AND 10),
    codigo_materia   TEXT        NOT NULL,
    actividad_id     UUID        REFERENCES actividades(id) ON DELETE SET NULL,
    is_rubric        BOOLEAN     NOT NULL DEFAULT FALSE,
    embed_model      TEXT        NOT NULL,
    embed_dimension  INT         NOT NULL,
    minio_path       TEXT        NOT NULL,
    original_filename TEXT       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX rag_curricular_scope_idx ON rag_curricular(carrera, semestre, codigo_materia);

-- Curricular chunk store (VECTOR(1536) = default AI_EMBED_DIMENSION)
CREATE TABLE rag_curricular_chunks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID        NOT NULL REFERENCES rag_curricular(id) ON DELETE CASCADE,
    chunk_index  INT         NOT NULL,
    chunk_text   TEXT        NOT NULL,
    embedding    VECTOR(1536),
    status       TEXT        NOT NULL CHECK (status IN ('pending', 'active')) DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX rag_curricular_chunks_doc_status_idx ON rag_curricular_chunks(document_id, status);
CREATE INDEX rag_curricular_chunks_vector_idx ON rag_curricular_chunks
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 10);

-- Docente knowledge base documents
CREATE TABLE rag_docente (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    docente_id       UUID        NOT NULL REFERENCES users(id),
    curso_id         UUID        NOT NULL REFERENCES cursos(id) ON DELETE CASCADE,
    actividad_id     UUID        REFERENCES actividades(id) ON DELETE SET NULL,
    is_rubric        BOOLEAN     NOT NULL DEFAULT FALSE,
    embed_model      TEXT        NOT NULL,
    embed_dimension  INT         NOT NULL,
    minio_path       TEXT        NOT NULL,
    original_filename TEXT       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX rag_docente_scope_idx ON rag_docente(docente_id, curso_id);

-- Docente chunk store (physically separate from rag_curricular_chunks — Principle V)
CREATE TABLE rag_docente_chunks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID        NOT NULL REFERENCES rag_docente(id) ON DELETE CASCADE,
    chunk_index  INT         NOT NULL,
    chunk_text   TEXT        NOT NULL,
    embedding    VECTOR(1536),
    status       TEXT        NOT NULL CHECK (status IN ('pending', 'active')) DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX rag_docente_chunks_doc_status_idx ON rag_docente_chunks(document_id, status);
CREATE INDEX rag_docente_chunks_vector_idx ON rag_docente_chunks
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 10);
