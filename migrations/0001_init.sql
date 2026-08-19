-- Enable the pgvector extension. Must run before any vector column is created.
CREATE EXTENSION IF NOT EXISTS vector;

-- users: bcrypt-hashed credentials, role-driven access.
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name          VARCHAR(100) NOT NULL,
    role          VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at    TIMESTAMP NOT NULL DEFAULT now(),
    updated_at    TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- knowledge_bases: a scoping boundary for documents and access control.
CREATE TABLE IF NOT EXISTS knowledge_bases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    team_id     UUID,
    created_by  UUID NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_kbs_deleted_at ON knowledge_bases(deleted_at);

-- documents: per-knowledge-base uploaded files and their index status.
CREATE TABLE IF NOT EXISTS documents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kb_id        UUID NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    file_path    VARCHAR(500) NOT NULL,
    file_size    BIGINT NOT NULL,
    file_type    VARCHAR(20) NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    chunk_count  INT NOT NULL DEFAULT 0,
    uploaded_by  UUID NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT now(),
    updated_at   TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_documents_kb_id ON documents(kb_id);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_deleted_at ON documents(deleted_at);

-- Full-text search index over document name for the search endpoint.
CREATE INDEX IF NOT EXISTS idx_documents_name_fts ON documents USING gin(to_tsvector('simple', name));

-- chunks: the vector-indexed text fragments used for semantic retrieval.
CREATE TABLE IF NOT EXISTS chunks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    doc_id       UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    content      TEXT NOT NULL,
    vector       vector(1536),
    chunk_index  INT NOT NULL,
    metadata     JSONB,
    created_at   TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_chunks_doc_id ON chunks(doc_id);
-- ivfflat gives fast approximate nearest-neighbour search over embeddings.
CREATE INDEX IF NOT EXISTS idx_chunks_vector ON chunks USING ivfflat (vector vector_cosine_ops) WITH (lists = 100);

-- qa_logs: every question asked and the answer produced, for history + dashboard.
CREATE TABLE IF NOT EXISTS qa_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL,
    kb_id         UUID NOT NULL,
    question      TEXT NOT NULL,
    answer        TEXT,
    sources       JSONB,
    feedback      VARCHAR(20) DEFAULT 'none',
    response_time INT,
    created_at    TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_qa_logs_user_id ON qa_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_qa_logs_kb_id ON qa_logs(kb_id);
CREATE INDEX IF NOT EXISTS idx_qa_logs_created_at ON qa_logs(created_at);
