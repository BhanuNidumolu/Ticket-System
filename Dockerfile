# ---- Build stage ----
FROM golang:1.23-alpine AS builder

WORKDIR /app

# No external modules are used, but copying go.mod first keeps the
# layer cache useful if that changes in the future.
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

# CGO_ENABLED=0 produces a fully static binary, so the runtime image
# can be a minimal scratch/alpine base with no libc dependency issues.
RUN CGO_ENABLED=0 GOOS=linux go build -o /ticket-system ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20

# Certs are needed if this service ever calls out over HTTPS; harmless
# to include now and saves a rebuild later.
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /ticket-system /app/ticket-system

EXPOSE 8080

ENTRYPOINT ["/app/ticket-system"]
