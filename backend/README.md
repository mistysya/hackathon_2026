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
- `DATABASE_DSN`: SQLite DSN. Its local default is `file:app.db?...`, so `go run ./cmd/api` works without a `/data` directory. Docker Compose overrides it to `/data/app.db` and retains WAL, foreign keys, and a 5000 ms busy timeout on each connection.

`schema.sql` is the only schema source. It is embedded into the binary and applied idempotently during startup.

## Parallel development boundary

The packages under `internal/domain`, `internal/ports`, and `internal/store`, plus `schema.sql`, are shared contracts. Feature worktrees should implement route registrars and services without changing the composition root in `cmd/api`; registrar wiring is reserved for the integration worktree.

For an endpoint slice, expose a `RegisterRoutes(chi.Router)` method on its handler (which satisfies `httpapi.RouteRegistrar`). The integration composition root passes that handler to `httpapi.NewRouter(logger, handler)`. This lets a slice such as Role B add endpoints without owning `cmd/api`, `go.mod`, or `go.sum`.
