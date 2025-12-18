# caiatech-datalab

description coming later

## Quick start (Docker)
From repo root:

```bash
cd caiatech-datalab
export DATALAB_ADMIN_TOKEN='change-me'
docker compose up -d --build
```

- Web UI: `http://localhost:3000`
- API: `http://localhost:8080`

Reset everything (including DB volume):

```bash
docker compose down -v
```

## Smoke test (API)

```bash
# health
curl -s http://localhost:8080/healthz

# submit a proposal (pending)
curl -s -X POST http://localhost:8080/api/v1/proposals \
  -H 'Content-Type: application/json' \
  -d '{"split":"train","messages":[{"role":"user","content":"hi"},{"role":"assistant","content":"hello"}]}'

# list pending proposals (admin)
curl -s 'http://localhost:8080/api/v1/proposals?status=pending' \
  -H "X-Admin-Token: $DATALAB_ADMIN_TOKEN"

# approve proposal id=1 (admin)
curl -s -X POST http://localhost:8080/api/v1/proposals/1/approve \
  -H "X-Admin-Token: $DATALAB_ADMIN_TOKEN"

# export as Onyx-style pairs JSONL
curl -s 'http://localhost:8080/api/v1/export.jsonl?type=pairs&split=train&status=approved&context=window&context_turns=6&role_style=labels' \
  | head
```

## Backend (local dev)

### Run Postgres

```bash
docker compose up -d postgres
```

### Run API
From `backend/`:

```bash
export DATALAB_DATABASE_URL='postgres://datalab:datalab@localhost:5432/datalab?sslmode=disable'
export DATALAB_ADMIN_TOKEN='change-me'

go run ./cmd/api
```

## Frontend (local dev, Vite + React)

From `frontend/`:

```bash
cp .env.example .env
npm install
npm run dev
```

Open `http://localhost:5173`.

## Key endpoints
- `GET /api/v1/conversations?split=train&status=approved&q=...`
- `GET /api/v1/conversations/{id}`
- `POST /api/v1/proposals` (submit conversation for review)
- `GET /api/v1/proposals?status=pending` (admin)
- `POST /api/v1/proposals/{id}/approve` (admin)
- `POST /api/v1/proposals/{id}/reject` (admin)
- `GET /api/v1/export.jsonl?...` (configurable)

### Export params
- `type=pairs|conversations`
- `split=train|valid|test|all`
- `status=approved|pending|draft|rejected|archived`
- `max_examples=0` (0 = unlimited)

Pairs-only params:
- `include_system=0|1`
- `context=none|window|full`
- `context_turns=6` (used when `context=window`)
- `role_style=labels|plain`

## Import JSONL (local)

Imports either:
- `{ "user": "...", "assistant": "..." }` (single-turn)
- `{ "messages": [{"role":"user"|"assistant"|"system","content":"..."}, ...] }` (canonical)

```bash
cd backend
export DATALAB_DATABASE_URL='postgres://datalab:datalab@localhost:5432/datalab?sslmode=disable'

# Import into a dataset (creates dataset if missing)
go run ./cmd/import_jsonl \
  --input /Users/owner/Desktop/caiatech/datasets/conversation/caia-chat.jsonl \
  --dataset 'caia-chat.jsonl' \
  --split train --status approved \
  --tags 'greetings,smalltalk' \
  --replace \
  --bad-out /Users/owner/Desktop/caiatech/datasets/conversation/caia-chat.bad.jsonl
```
# caiatech-datalab
