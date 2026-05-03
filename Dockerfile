# ─── Stage 1: Build ───────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# git needed for go mod download with VCS stamping
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependency layer separately from source
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -trimpath \
    -o fuzzyrouter \
    ./cmd/fuzzyrouter

# ─── Stage 2: Final (scratch) ─────────────────────────────────────────────────
FROM scratch

# TLS roots for outbound HTTPS redirect targets
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Timezone data (optional but avoids slog timestamp issues in some locales)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /build/fuzzyrouter /fuzzyrouter

EXPOSE 8080

# Config file expected at /config.yaml (mount via volume or override FUZZY_CONFIG)
ENV FUZZY_CONFIG=/config.yaml

ENTRYPOINT ["/fuzzyrouter"]
