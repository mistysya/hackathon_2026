# Docker / Colima validation

Validated on Colima's Linux arm64 VM (2 CPUs, 2 GiB RAM), Docker Engine 28.3.3, Compose 2.39.2, Go 1.27.1 containers, and host Chromium via Playwright. The default Docker context was not changed.

## Results

- Compose development image build: passed.
- `go test ./...`, `go test -race ./...`, and `go vet ./...` inside the Compose container: passed.
- Standalone runtime image build: passed.
- **Found and fixed a runtime startup bug:** the image ran as a non-root user in root-owned `/app`, while the default relative SQLite DSN tried to write there. Startup failed with `unable to open database file (14)`. The runtime now defaults `DATABASE_DSN` to `/data/app.db`, with WAL, foreign keys, and busy timeout retained.
- Confirmed the runtime runs as **UID 10001**, without an environment override, and creates its database successfully in the named volume.
- Browser/API suite against the **Compose container on port 18081: 6/6 passed** (two scenarios repeated three times).
- Browser/API suite against the **runtime image on port 18080: 6/6 passed**.
- Both suites exercise import → profile → generate → approve → simulate → actual email CTA → Go landing → training → dashboard, plus rejection, validation, privacy, deduplication, and concurrent simulation checks.
- After recreating **both** containers with their existing named volumes, completed reports and landing tokens remained available and unchanged.
- Runtime shutdown via `docker stop` completed with exit code **0**.

Port 8080 was occupied by an existing native API process. Compose was moved to 18081 rather than stopping that process. The initial browser run on 8080 was not counted as Docker evidence; the container-backed endpoint was verified before rerunning. `API_PORT` now makes the Compose host port configurable.

## Reproduce: Compose + browser tests

Use a new project name to isolate test volumes. These tests intentionally write fictional employees and campaigns, so **do not point them at a production or valuable demo database**.

```sh
colima start --activate=false
cd backend
docker --context colima compose -p hackathon-docker-check build api
docker --context colima compose -p hackathon-docker-check run --rm api \
  sh -c 'go test ./... && go test -race ./... && go vet ./...'
API_PORT=18081 docker --context colima compose -p hackathon-docker-check up -d api
# Wait until ready; inspect Compose logs if this fails.
curl --fail http://127.0.0.1:18081/employees

cd ../frontend
npm ci
npx playwright install chromium
E2E_API_URL=http://127.0.0.1:18081 npm test -- --project real-api --repeat-each=3
```

`E2E_API_URL` tells Playwright to use the already-running container API instead of starting Go on the host. Vite and Chromium still run locally, testing through the same frontend proxy as the demo. Without this variable, tests retain their isolated native Go/in-memory SQLite behavior.

## Reproduce: runtime image + browser tests

```sh
cd backend
docker --context colima build --target runtime -t hackathon-docker-check-runtime .
docker --context colima run -d --name hackathon-docker-check-runtime \
  -p 127.0.0.1:18080:8080 \
  --mount type=volume,source=hackathon-docker-check-runtime-data,target=/data \
  hackathon-docker-check-runtime
docker --context colima exec hackathon-docker-check-runtime id
curl --fail http://127.0.0.1:18080/employees

cd ../frontend
E2E_API_URL=http://127.0.0.1:18080 npm test -- --project real-api --repeat-each=3
```

To check persistence, save a campaign ID and its report, stop/remove the runtime container, then run the same `docker run` command with the same named volume. The report and `/landing/{token}` should still work. For Compose, `up -d --force-recreate api` retains the project volume.

## Cleanup (test resources only)

```sh
cd backend
API_PORT=18081 docker --context colima compose -p hackathon-docker-check down --volumes
docker --context colima rm -f hackathon-docker-check-runtime
docker --context colima volume rm hackathon-docker-check-runtime-data
```

These commands delete the named test databases and Compose caches, not other projects' volumes. Do not substitute a project name containing data you need to keep.

## Scope

This verifies Linux arm64 under Colima, not amd64 or every container runtime. The frontend is run on the host (there is no frontend Dockerfile). Live LLM/search/email integrations remain outside this fixture-backed demo.
