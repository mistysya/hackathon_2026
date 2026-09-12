FROM golang:1.27.1-bookworm AS dev

WORKDIR /workspace/backend

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["go", "run", "./cmd/api"]

FROM dev AS test
RUN go test ./...

FROM dev AS build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM debian:bookworm-slim AS runtime
RUN useradd --system --uid 10001 --create-home api

RUN mkdir -p /data && chown api:api /data

WORKDIR /app
COPY --from=build /out/api /usr/local/bin/api

USER api
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
