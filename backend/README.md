# Go Backend Foundation

This directory contains the Go, HTTP, and SQLite backend for the hackathon MVP. It includes the fixture-only employee import/query, profile enrichment, and campaign generation workflow.

## Run locally

Requires Go 1.27.1+. From this directory:

```sh
go run ./cmd/api
go test -race ./...
go vet ./...
```

For the complete local frontend/backend demo, run `bash scripts/demo.sh` from the repository root and follow [the demo guide](../README.md).

## Run with Docker

```sh
docker compose up --build api
```

The server listens on `http://localhost:8080`. Set `API_PORT=18081` before the Compose command to use a different host port without changing the container's listen address.

On macOS with Colima, run `colima start --activate=false`, then use `docker --context colima ...` for the commands above and below. This avoids changing your default Docker context. See [Docker validation](../docs/docker-validation.md) for isolated Compose/runtime browser tests and persistent-volume checks.

The integrated endpoints are:

- `POST /employees/import`, `GET /employees`, and `GET /employees/{id}`
- `POST /employees/{id}/enrich`
- `POST /campaigns/generate`, `GET /campaigns/{id}`, `POST /campaigns/{id}/approve`, and `POST /campaigns/{id}/reject`
- `POST /campaigns/{id}/simulate`
- `GET /landing/{token}` and `POST /events`
- `GET /reports/{campaignId}`

Together these endpoints provide the complete fixture-only training flow from employee import through reporting.

## Verify

```sh
docker compose run --rm api go test ./...
docker compose run --rm api go test -race ./...
docker compose run --rm api go vet ./...
docker build --target runtime -t hackathon-2026-api .
```

## Standalone non-root runtime image

```sh
docker build --target runtime -t hackathon-2026-api .
docker run --rm --name hackathon-api -p 127.0.0.1:8080:8080 \
  --mount type=volume,source=hackathon-api-data,target=/data \
  hackathon-2026-api
```

The runtime runs as UID 10001 and defaults SQLite to `/data/app.db`. The named volume retains employees, campaigns, tokens, and reports when the container is recreated. No `DATABASE_DSN` override is necessary. Bind-mounted host directories, if used instead, must be writable by UID 10001.

## Configuration

- `HTTP_ADDR`: listen address, default `:8080`.
- `API_PORT`: Docker Compose's localhost host port, default `8080`.
- `DATABASE_DSN`: SQLite DSN. Its local default is `file:app.db?...`, so `go run ./cmd/api` works without a `/data` directory. Docker Compose and the runtime image default it to `/data/app.db` and retains WAL, foreign keys, and a 5000 ms busy timeout on each connection.

`schema.sql` is the only schema source. It is embedded into the binary and applied idempotently during startup.

## Parallel development boundary

The packages under `internal/domain`, `internal/ports`, and `internal/store`, plus `schema.sql`, are shared contracts. Feature worktrees should implement route registrars and services without changing the composition root in `cmd/api`; registrar wiring belongs in the integration worktree.

For an endpoint slice, expose a `RegisterRoutes(chi.Router)` method on its handler (which satisfies `httpapi.RouteRegistrar`). The integration composition root passes that handler to `httpapi.NewRouter(logger, handler)`. This lets a slice such as Role B add endpoints without owning `cmd/api`, `go.mod`, or `go.sum`.
