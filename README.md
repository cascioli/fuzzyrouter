# FuzzyRouter

[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Docker Image Size](https://img.shields.io/badge/image%20size-%3C10MB-brightgreen?logo=docker)](Dockerfile)
[![Go Report Card](https://goreportcard.com/badge/github.com/simlimone/fuzzyrouter)](https://goreportcard.com/report/github.com/simlimone/fuzzyrouter)

**FuzzyRouter** is a lightweight, zero-config-friendly Docker microservice that acts as a smart HTTP catch-all. When a user hits a mistyped subdomain — `atp.example.com`, `adnin.example.com` — FuzzyRouter calculates string similarity against your list of known subdomains and issues a `301`/`302` redirect to the closest match.

```
User types:   atp.example.com
              ↓
         FuzzyRouter
              ↓  (Levenshtein: "atp" → "app", score 0.667)
Redirects to: app.example.com
```

---

## How It Works

### The Catch-All DNS Concept

Normally, mistyped subdomains return an NXDOMAIN error. FuzzyRouter fixes that in two steps:

1. **Wildcard DNS record** — Add a `*.example.com → <your-server>` DNS entry. All subdomains without a specific record now resolve to your server.
2. **FuzzyRouter as catch-all** — Run FuzzyRouter on that server. It intercepts every request, extracts the subdomain from the `Host` header, and finds the best match from your configured list using [Levenshtein distance](https://en.wikipedia.org/wiki/Levenshtein_distance).

```
DNS:
  app.example.com     → 1.2.3.4   (your real app server — specific record)
  *.example.com       → 1.2.3.4   (catch-all → FuzzyRouter port)

Request flow:
  atp.example.com  →  FuzzyRouter  →  302 → app.example.com
  xyz.example.com  →  FuzzyRouter  →  404 (score below threshold)
```

### Matching Algorithm

FuzzyRouter uses normalized Levenshtein distance:

```
score = 1 - (edit_distance / max(len(a), len(b)))
```

A score of `1.0` is an exact match. Requests below `match_threshold` (default `0.5`) return HTTP 404 instead of a redirect.

| Input    | Match   | Score |
|----------|---------|-------|
| `atp`    | `app`   | 0.667 |
| `adnin`  | `admin` | 0.800 |
| `apii`   | `api`   | 0.750 |
| `blg`    | `blog`  | 0.750 |
| `xyz`    | —       | < 0.5 → 404 |

---

## Quickstart

### Prerequisites

- Docker & Docker Compose v2

### 1. Clone

```bash
git clone https://github.com/simlimone/fuzzyrouter.git
cd fuzzyrouter
```

### 2. Configure

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` to match your domain and subdomains:

```yaml
base_domain: example.com
subdomains:
  - app
  - admin
  - api
  - auth
  - blog
```

### 3. Run

```bash
docker compose up --build
```

### 4. Test

```bash
# Should redirect to http://app.example.com/
curl -v http://localhost:8080/ -H "Host: atp.example.com"

# Health probe
curl http://localhost:8080/healthz
# → {"status":"ok"}
```

---

## Configuration

FuzzyRouter reads configuration from a YAML file. **Environment variables override file values** — useful for Docker secrets, Kubernetes ConfigMaps, or CI.

### YAML (`config.yaml`)

```yaml
# TCP port to listen on
port: 8080

# Log verbosity: debug | info | warn | error
log_level: info

# HTTP redirect code: 301 (permanent) or 302 (temporary)
redirect_code: 302

# Root domain — appended to every matched subdomain
base_domain: example.com

# Minimum similarity score to accept a match [0.0–1.0]
# Below this score → HTTP 404 instead of redirect
match_threshold: 0.5

# Exhaustive list of valid subdomains
subdomains:
  - app
  - admin
  - api
  - auth
  - blog
  - shop
  - docs
  - status
  - mail
```

### Environment Variables

| Variable              | Default        | Description                            |
|-----------------------|----------------|----------------------------------------|
| `FUZZY_CONFIG`        | `config.yaml`  | Path to the YAML config file           |
| `FUZZY_PORT`          | `8080`         | Listening port                         |
| `FUZZY_LOG_LEVEL`     | `info`         | Log level (`debug/info/warn/error`)    |
| `FUZZY_REDIRECT_CODE` | `302`          | HTTP redirect code (`301` or `302`)    |
| `FUZZY_BASE_DOMAIN`   | —              | Base domain (required)                 |
| `FUZZY_SUBDOMAINS`    | —              | Comma-separated subdomain list         |
| `FUZZY_THRESHOLD`     | `0.5`          | Match threshold `[0.0–1.0]`            |

**Env-only example** (no config file):

```bash
docker run --rm \
  -e FUZZY_BASE_DOMAIN=example.com \
  -e FUZZY_SUBDOMAINS=app,admin,api \
  -e FUZZY_LOG_LEVEL=debug \
  -p 8080:8080 \
  fuzzyrouter:dev
```

---

## Structured Logging

FuzzyRouter outputs JSON logs to stdout:

```json
{"time":"2024-01-15T10:23:01Z","level":"INFO","msg":"server starting","addr":":8080","base_domain":"example.com"}
{"time":"2024-01-15T10:23:05Z","level":"INFO","msg":"redirect","from":"atp","to":"app","score":"0.667","target":"http://app.example.com/","method":"GET","remote_addr":"172.18.0.1:54321","user_agent":"curl/8.4.0"}
{"time":"2024-01-15T10:23:07Z","level":"WARN","msg":"no match found","subdomain":"xyz","remote_addr":"172.18.0.1:54322"}
```

---

## DNS Setup

Add these two records in your DNS provider:

| Type  | Name            | Value      | TTL |
|-------|-----------------|------------|-----|
| A     | `app.example.com`   | `1.2.3.4`  | 300 |
| A     | `*.example.com`     | `1.2.3.4`  | 300 |

> **Note:** Specific A/AAAA records take priority over the wildcard. Your real services remain unaffected.

---

## Production Deployment

FuzzyRouter is typically placed behind a reverse proxy (nginx, Traefik, Caddy):

```
Internet → Reverse Proxy (:443) → FuzzyRouter (:8080)
```

The `X-Forwarded-Proto` header is respected automatically — redirects will use `https://` when the proxy sets it.

**Traefik label example:**

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.fuzzyrouter.rule=HostRegexp(`{subdomain:.+}.example.com`)"
  - "traefik.http.routers.fuzzyrouter.priority=1"  # lowest — catch-all
```

---

## Project Structure

```
fuzzyrouter/
├── cmd/fuzzyrouter/        # Entry point
│   └── main.go
├── internal/
│   ├── config/             # YAML + ENV config loader
│   │   ├── config.go
│   │   └── config_test.go
│   ├── matcher/            # Fuzzy matching (interface-driven)
│   │   ├── matcher.go      # Matcher interface + Result type
│   │   ├── levenshtein.go  # Levenshtein implementation
│   │   └── levenshtein_test.go
│   └── server/             # HTTP handler + lifecycle
│       ├── server.go
│       └── server_test.go
├── config.example.yaml     # Annotated sample config
├── Dockerfile              # Multi-stage, scratch final image
├── docker-compose.yml
└── go.mod
```

---

## Extending the Matcher

`matcher.Matcher` is an interface — swap in any algorithm:

```go
type Matcher interface {
    Match(input string) (match string, score float64)
}
```

Example: add a Jaro-Winkler matcher and pass it to `server.Options.Matcher`. No other code changes required.

---

## License

MIT — see [LICENSE](LICENSE).
