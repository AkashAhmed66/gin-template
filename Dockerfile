# syntax=docker/dockerfile:1

# =============================================================================
# Stage 1 — builder
# Compiles a static Linux binary and generates the Swagger spec at build time.
# =============================================================================
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux

RUN apk add --no-cache git ca-certificates

# swag CLI generates docs/docs.go which main.go blank-imports at build time.
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6

WORKDIR /src

# Cache module downloads — only invalidated when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Source.
COPY . .

# Generate the Swagger spec into docs/.
RUN swag init --parseDependency --parseInternal

# Trimpath + strip symbols → smaller binary, no host-path leakage.
RUN go build -trimpath -ldflags="-s -w" -o /out/server .


# =============================================================================
# Stage 2 — runtime
# Alpine for a tiny image with a shell + wget for healthchecks. ~25 MB final.
# =============================================================================
FROM alpine:3.19

# ca-certificates: outbound TLS (Postgres, SMTP, webhooks)
# tzdata:          IANA timezone DB for cron / lumberjack LocalTime
# wget:            HEALTHCHECK probe
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S app && \
    adduser -S app -G app

WORKDIR /app

# Binary + runtime assets the app reads from disk.
COPY --from=builder /out/server         ./server
COPY --from=builder /src/migrations     ./migrations
COPY --from=builder /src/templates      ./templates
COPY --from=builder /src/docs           ./docs

# Mutable directories the app writes to. Volume-mount these in compose/k8s.
RUN mkdir -p /app/uploads /app/logs && chown -R app:app /app

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget --spider --quiet http://localhost:8080/health || exit 1

ENTRYPOINT ["./server"]
