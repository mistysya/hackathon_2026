# Go Backend Foundation

This directory contains the shared Go, HTTP, and SQLite foundation for the hackathon backend. Business endpoints are intentionally not registered at this stage.

## Run with Docker

```sh
docker compose up --build api
```

The server listens on `http://localhost:8080`. Until feature route registrars are integrated, requests return the frozen JSON error envelope with `404 not_found`.

## Verify

```sh
docker compose run --rm api go test ./...
docker compose run --rm api go test -race ./...
docker compose run --rm api go vet ./...
docker build --target runtime -t hackathon-2026-api .
```

## Configuration

- `HTTP_ADDR`: listen address, default `:8080`.
- `DATABASE_DSN`: SQLite DSN. The container default stores data at `/data/app.db` and enables WAL, foreign keys, and a 5000 ms busy timeout on each connection.

`schema.sql` is the only schema source. It is embedded into the binary and applied idempotently during startup.

## Parallel development boundary

The packages under `internal/domain`, `internal/ports`, and `internal/store`, plus `schema.sql`, are shared contracts. Feature worktrees should implement route registrars and services without changing the composition root in `cmd/api`; registrar wiring is reserved for the integration worktree.
