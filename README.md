# Housing-Agent

A Go service that answers questions about Lagos State tenancy law using retrieval (pgvector in Postgres) + an LLM via OpenRouter.

## What It Does
- Seeds a `documents` table with text chunks (currently from `internal/data/data.go`)
- Generates embeddings for chunks and stores them in Postgres (pgvector)
- Exposes an HTTP API to search and answer questions using the retrieved context

## API

### Health
`GET /health`

### Search
`GET /search?query=...`

Example:
```bash
curl "http://localhost:8080/search?query=tenancy%20agreement"
```

## Local Setup

### 1) Start Postgres with pgvector
Docker must be running (Docker Desktop or the Docker daemon).
This repo includes a compose file.

```bash
docker compose up -d --build
```

### 2) Configure environment
Create a `.env` file (do not commit real keys) with at least:

```env
OPENROUTER_API_KEY=...
DB_ADDR=postgres://admin:adminpassword@localhost:5433/housing_agent_db?sslmode=disable
ADDR=:8080
```

Optional (defaults exist in code):
```env
OPENROUTER_API_BASE_URL=https://openrouter.ai/api/v1
OPENROUTER_API_MODEL=google/gemini-embedding-2-preview
```

### 3) Run the server
Air is supported (recommended for local dev):
```bash
air
```

Or run without Air:
```bash
go run ./cmd
```

### 4) Confirm the database is up
The app expects Postgres to be reachable at `DB_ADDR` and the database to exist (via compose it’s `housing_agent_db`).

Quick check:
```bash
docker compose ps
```



## Notes
- The embedding column uses Postgres `vector` type (pgvector). Model embedding dimensions can vary by provider/model.
- Seeding runs on startup and is intended for a single prefilled dataset.

## Future Improvements (Document Upload + Processing)
Goal: allow uploading any document (PDF/DOCX/TXT), chunk it, embed it, and make it searchable.

### Suggested Architecture
- **Upload endpoint**
  - `POST /documents` (multipart/form-data) with file + optional metadata (title/source)
  - Store raw file (local disk, S3, or a blob table)
- **Text extraction**
  - PDF: extract text per page
  - DOCX: extract paragraphs
  - TXT: read directly
- **Chunking pipeline**
  - Split by headings/sections when possible
  - Fallback: fixed-size chunks (e.g., 500–1,000 tokens) with overlap
  - Store chunk metadata: `source`, `page`, `section`, `title`
- **Async processing**
  - Upload returns `202 Accepted` with a job id
  - Background worker performs extraction → chunking → embedding → insert into DB
- **Multi-document search**
  - Add filtering by `source` or `document_id`
  - Add indexes (ivfflat/hnsw) for pgvector for faster similarity queries
- **Schema changes**
  - `documents` table becomes `chunks`
  - Add `document_files` table to track uploads and processing status
  - Add `document_id` foreign key on chunks
- **Security**
  - File size limits, content-type validation, antivirus scanning (optional)
  - Authentication for uploads and access control per user/tenant
