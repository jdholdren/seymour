# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Seymour?

Seymour is a single-tenant RSS feed aggregator with AI-powered curation. Users subscribe to RSS feeds, and a Temporal worker syncs feeds, builds a timeline, then uses the Anthropic Claude API to judge/curate entries based on a user-defined prompt.

This is a monorepo: the Go backend lives under `server/` and the Vue/Vite frontend lives under `fe/`. `docker-compose.yaml`, the `Makefile`, and CI workflows live at the repo root and orchestrate both.

## Common Commands

- `make test` — Run all Go tests (`go -C server test ./...`)
- `go -C server test -run TestName ./internal/path/...` — Run a single test
- `make up` — Start all services via docker compose (worker, api, fe, temporal-server)
- `make build` — Build all Docker images
- `make rb-api` — Rebuild and restart only the API service
- `make rb-worker` — Rebuild and restart only the worker service

## Architecture

Two binaries, both in `server/cmd/`:

- **`server/cmd/api`** — REST API server (port 4444). HTTP handlers in `server/internal/api/`. Uses Gorilla Mux for routing.
- **`server/cmd/worker`** — Temporal workflow worker. Workflows and activities in `server/internal/worker/`.

The frontend (`fe/`) is a Vue/Vite SPA served via nginx in production (port 3000 in docker compose, mapped to container port 80), and talks to the API directly over `VITE_API_HOST` (no reverse proxy).

### Core packages (under `server/`)

- **`internal/seymour`** — Domain models and the `Repository` interface. All data access goes through this single interface. `DBTime` is a custom type for SQLite datetime marshaling using RFC3339.
- **`internal/sqlite`** — SQLite implementation of `Repository`. Uses `sqlx` + `squirrel` query builder. Pure-Go SQLite driver (no CGO): `modernc.org/sqlite`.
- **`internal/sync`** — RSS feed parsing and sync logic. Parses XML, sanitizes HTML, extracts feed metadata.
- **`internal/worker`** — Temporal workflows and activities:
  - `SyncAllFeeds` — Scheduled every 15 min, batches feeds in groups of 50
  - `CreateFeed` — Creates feed, syncs, rolls back on failure
  - `RefreshTimeline` — Inserts missing timeline entries, triggers judging
  - `JudgeTimeline` — Calls Claude API to approve/reject entries (batches of 20, max 3 loops)
- **`internal/migrations`** — Embedded SQL migration files, run via `golang-migrate`
- **`internal/errors`** — Custom error type with HTTP status codes; wraps as non-retryable Temporal errors for internal failures

### Temporal patterns

- Task queue name: `shared`
- Singleton workflows use `WorkflowIDReusePolicy: TERMINATE_IF_RUNNING`
- Schedules: `sync_all` and `refresh_timelines` both run every 15 minutes
- Child workflows use `ParentClosePolicy: ABANDON` so they outlive parents

### ID generation

UUIDs with namespace suffixes: e.g. `{uuid}-fd` for feeds. See the `internal/sqlite` package.

### Database

SQLite with connection flags `-txlock=immediate -busy_timeout=5000`. Migrations are embedded Go files. Timeline entry statuses: `requires_judgement`, `approved`, `rejected`.

## Environment Variables

**API:** `DATABASE` (SQLite path), `TEMPORAL_HOST_PORT`, `PORT` (default 4444), `CORS`
**Worker:** `DATABASE`, `TEMPORAL_HOST_PORT`, `CLAUDE_API_KEY`

API key is injected via `.env` file in docker-compose.

## API Endpoints

- `GET /api/viewer` — Viewer info
- `GET/PUT /api/prompt` — Active curation prompt
- `POST /api/subscriptions` — Subscribe to feed (triggers CreateFeed workflow)
- `GET /api/subscriptions` — List subscriptions
- `GET /api/timeline` — Paginated curated timeline (supports `feed_id` filter)
- `GET /api/feed-entries/{feedEntryID}` — Full article content via go-readability
