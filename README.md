# 企业级知识库问答系统 (Enterprise Knowledge Q&A System)

A dockerized enterprise knowledge-base Q&A system. Upload documents (PDF / Word / Markdown / text),
they are parsed, chunked, embedded, and stored in PostgreSQL+pgvector; ask questions and get
answers grounded in your knowledge with source citations.

The whole stack starts with one command and runs **without any external API keys** — LLM and
embedding providers default to in-process mock implementations so you can verify the full pipeline
immediately. Flip a couple of environment variables to switch to OpenAI (or any OpenAI-compatible
endpoint) and MinIO object storage.

## Quick start

```bash
cd backend
docker compose up -d --build
```

- App:        http://localhost:8088  (API + the React SPA)
- MinIO UI:   http://localhost:9001  (minioadmin / minioadmin)
- PostgreSQL: localhost:5432 (kn / kn)

> The host port is 8088 to avoid clashing with anything already on 8080. Change it in
> `backend/docker-compose.yml` (`ports`) if you prefer 8080.

Open the app in a browser, register an account (pick the admin role), create a knowledge base,
upload a document, wait for "已索引", then ask a question in the Q&A page. The answer streams in
token-by-token with the retrieved source snippets shown beneath it.

## Architecture

The backend is structured for single-responsibility: each package owns one concern and depends on
narrow interfaces, so storage / embedder / LLM / queue are all swappable.

```
backend/
  cmd/server/main.go            # bootstrap: config, logging, signals, graceful shutdown
  internal/
    config/                     # all runtime config (viper + explicit env bindings)
    server/                     # composition root: wires every dependency + route table
    middleware/                 # auth (JWT), recover, request log, role gating
    handler/                    # HTTP adapters — parse request, call service, map errors
    service/                    # business rules: auth, kb, document, indexing, qa, dashboard
    repository/                 # persistence (GORM) + pgvector retrieval + migrations runner
    queue/                      # async indexing queue (in-memory, retry x3)
    storage/                    # Storage interface: local | minio
    embedder/                   # Embedder interface: mock | openai
    llm/                        # LLM interface: mock | openai
    parser/                     # document parsers: pdf / docx / plain + the chunker
    auth/                       # JWT issue/verify + bcrypt password
    web/                        # serves the compiled React SPA (SPA fallback router)
    model/                      # GORM domain entities (users, kb, documents, chunks, qa_logs)
    errors/                     # classified app errors -> HTTP status/code mapping
  pkg/
    openai/                     # shared OpenAI-compatible HTTP client
    logger/                     # zap singleton
    response/                   # unified JSON envelope + error helpers
migrations/0001_init.sql        # schema + pgvector extension + indexes
```

### Request flow (question answering)

```
question -> QAService
  -> embed question (Embedder)
  -> pgvector cosine search top-k chunks (DocumentRepo.RetrieveByVector)
  -> assemble citation block + system+user prompt
  -> stream LLM tokens back over SSE
  -> persist QALog (question, answer, sources, latency)
```

Indexing is fully asynchronous: `POST /kb/:id/documents` stores the file and enqueues a job, then
returns immediately. A worker parses, chunks, embeds, and persists; the document's status flips
`pending → indexing → indexed` (or `failed`, retried 3×), polled by the frontend.

### Swapping providers (non-functional requirement: extensibility)

All providers are interface-based; pick via environment variables:

| Concern   | Default  | Switch to          | Env vars                                              |
|-----------|----------|--------------------|-------------------------------------------------------|
| Storage   | local    | MinIO / S3         | `STORAGE_PROVIDER=minio`, `STORAGE_ENDPOINT`, `STORAGE_ACCESS_KEY`, `STORAGE_SECRET_KEY`, `STORAGE_BUCKET` |
| Embedder  | mock     | OpenAI / BGE-via-gateway | `EMBEDDER_PROVIDER=openai`, `EMBEDDER_API_KEY`, `EMBEDDER_MODEL`, `EMBEDDER_DIM` |
| LLM       | mock     | OpenAI / Claude (compatible) | `LLM_PROVIDER=openai`, `LLM_API_KEY`, `LLM_MODEL`, `LLM_BASE_URL` |
| Queue     | memory   | redis (interface ready) | `QUEUE_PROVIDER=redis` (implementation stub)           |

Set these in `backend/docker-compose.yml`'s `app.environment` block.

## Local development (without Docker)

Backend:
```bash
cd backend
go mod tidy
# start just the infra: docker compose up -d postgres redis minio
APP_ENV=development go run ./cmd/server
```

Frontend (hot reload):
```bash
cd frontend
npm install
npm run dev      # http://localhost:5173, proxies /api to :8080
```

## API summary

All under `/api/v1`, all but `auth/register` `auth/login` require `Authorization: Bearer <jwt>`.

| Method | Path | Description |
|--------|------|-------------|
| POST | /auth/register | register, returns user+token |
| POST | /auth/login    | login, returns user+token |
| GET  | /auth/me       | current user |
| POST | /kb            | create knowledge base |
| GET  | /kb            | list knowledge bases |
| GET  | /kb/:id        | get one |
| PUT  | /kb/:id        | update |
| DELETE | /kb/:id       | delete (admin) |
| POST | /kb/:id/invite | invite member (stub) |
| POST | /kb/:id/documents | upload document (multipart) |
| GET  | /kb/:id/documents | list documents |
| GET  | /documents/:id | document detail |
| GET  | /documents/:id/status | index status (poll) |
| DELETE | /documents/:id | delete |
| GET  | /documents/:id/download | download file |
| POST | /documents/:id/reindex | retry indexing |
| GET  | /search?kbId=&q= | keyword search |
| POST | /qa/ask        | ask (SSE if `Accept: text/event-stream`, else JSON) |
| GET  | /qa/history    | my Q&A history |
| POST | /qa/feedback   | up/down vote |
| GET  | /dashboard     | ops metrics (manager+) |

Every response uses one envelope: `{ "success": bool, "data"?: T, "error"?: { code, message } }`.

## Security notes (implemented)

- bcrypt password hashing; raw passwords never persisted.
- JWT (HS256), configurable expiry (default 24h), verified per request.
- Role-based access: member / manager / admin, enforced at route + handler level.
- Panic recovery middleware → 500 envelope, never a stack trace to the client.
- LLM/embedding failures degrade to a friendly message rather than leaking provider errors.
- File downloads are scoped by document id; traversal is blocked in the local storage path.

## Notes / trade-offs

- **Mock LLM & embedder**: the mock embedder is a deterministic hash vector — it exercises the full
  retrieval/storage pipeline (vectors are stored, cosine search runs) but is not semantically
  meaningful. Switch to `openai` for real quality. The mock LLM echoes the question and notes the
  provider is a mock, so the SSE + citation UI is verifiable without a key.
- **Queue**: in-memory, not durable (a restart drops pending jobs). The `Queue` interface is ready
  for a Redis-Stream implementation; the `redis` provider is a documented stub.
- **Migrations**: applied on startup by reading `/migrations/*.sql` in order; files use
  `CREATE ... IF NOT EXISTS` so they're idempotent. A migrations-tracking table can be added without
  changing call sites when this grows.
