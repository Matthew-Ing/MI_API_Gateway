# MI API Gateway

A Go reverse proxy in front of two mock services. nginx terminates TLS at the edge; the gateway applies auth (JWT or API key), Redis rate limiting, and a per-upstream circuit breaker, then proxies with `httputil.ReverseProxy`. Routing is YAML, not a framework mux per backend.

Built with Go `net/http` — no chi, gin, or echo.

## Architecture

```mermaid
flowchart LR
  client[Client] --> nginx
  adminUI[Admin UI] --> nginx
  nginx -->|"/api/"| gateway[Go gateway]
  nginx -->|static| adminUI
  gateway --> redis[Redis]
  gateway --> users[User service]
  gateway --> orders[Order service]
  gateway --> jaeger[Jaeger OTLP]
  prometheus[Prometheus] --> gateway
  grafana[Grafana] --> prometheus
```

Request path: **TLS / static at nginx** → **auth → rate limit → circuit breaker → reverse proxy** on the gateway. Redis stores hashed API keys and per-identity counters. `/healthz` and `/metrics` skip the policy chain.

## Features

- YAML route table: prefix match (`/users` covers `/users` and `/users/1`)
- JWT (`Authorization: Bearer`) or API key (`X-API-Key`)
- API keys stored as SHA-256 hashes in Redis (`apikey:<hex>`); Compose seeds `demo-key` via `SAMPLE_API_KEY`
- Fixed-window rate limit in Redis (`429` + `Retry-After`)
- Per-upstream circuit breaker (closed / open / half-open) → `503` when open
- Middleware: request ID, access log, panic recover, OpenTelemetry, Prometheus
- Admin API + Vite/React SPA: login, mint/revoke keys, upstream health and breaker state
- nginx edge: HTTPS, security headers, gzip, `/api/` → gateway, `/admin/` → SPA
- Docker Compose: gateway, Redis, mocks, nginx, Jaeger, Prometheus, Grafana
- Kustomize manifests under `deploy/k8s/`
- GitHub Actions: `go test`, golangci-lint, gateway image build

## Quick start

From the repo root:

```powershell
docker compose -f deploy/compose.yml up --build
```

| Surface | URL |
|---|---|
| nginx HTTPS (preferred) | `https://localhost:8443` (HTTP `:8085` redirects here) |
| Gateway (direct) | `http://localhost:8089` |
| Admin UI | `https://localhost:8443/admin/` |
| Jaeger | `http://localhost:16686` |
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` (anonymous Admin) |

nginx strips `/api/` before proxying, so gateway `/users` is `https://localhost:8443/api/users`. Self-signed TLS: `-k` with curl.

```powershell
curl.exe -k https://localhost:8443/api/healthz
# status ok
```

Unauthenticated proxied routes are rejected:

```powershell
curl.exe -k -i https://localhost:8443/api/users
# HTTP/1.1 401 Unauthorized
```

### API key

Compose seeds `demo-key` at gateway startup. Send the raw key on the request.

```powershell
curl.exe -k -H "X-API-Key: demo-key" https://localhost:8443/api/users
curl.exe -k -H "X-API-Key: demo-key" https://localhost:8443/api/users/1
curl.exe -k -H "X-API-Key: demo-key" https://localhost:8443/api/orders
```

Mint extra keys from the admin UI, or hash a custom key and `SET` it in Redis (`apikey:<hex>` = `1`).

```powershell
python -c "import hashlib; print(hashlib.sha256(b'your-key').hexdigest())"
```

### JWT

Compose sets `JWT_SECRET=secret`. Tokens are HS256 with a `userID` claim.

```powershell
curl.exe -k -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySUQiOiJhbGljZSJ9.Qv5mGwg-1oJrJCGaUhrJ9mleOFdEFHrGscn5-5p7dnk" https://localhost:8443/api/users
```

Mint your own with the same secret (HS256, payload `{"userID":"..."}`).

### Rate limit

Compose sets **100 requests per 60 seconds** per identity (API key hash, Bearer token hash, or client address). The process default without env is 10. When the window is exceeded:

```text
HTTP/1.1 429 Too Many Requests
Retry-After: 60
```

```powershell
1..12 | ForEach-Object {
  curl.exe -k -s -o NUL -w "%{http_code}`n" -H "X-API-Key: demo-key" https://localhost:8443/api/users
}
```

### Circuit breaker

The order mock uses `FAIL_RATE=0.8` in Compose so many `/orders` calls return 5xx. After **10 failures** the breaker opens for **10 seconds** and the gateway returns `503` without hitting the upstream. Watch `gateway_circuit_state` / `gateway_circuit_open_total` in Grafana, or `GET /admin/health` after logging in.

### Admin UI

Open `https://localhost:8443/admin/` (Compose password is `admin`). Login issues an admin JWT (`ADMIN_JWT_SECRET`). The SPA calls `/api/admin/*` through nginx:

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/admin/login` | Password → `{ "token": "..." }` |
| `POST` | `/admin/keys` | Mint a key (`Authorization: Bearer` admin token) |
| `DELETE` | `/admin/keys/{hash}` | Revoke |
| `GET` | `/admin/health` | Upstream reachability + breaker state |

## Routes

From [`configs/gateway.yaml`](configs/gateway.yaml):

| Path prefix | Upstream | Service |
|---|---|---|
| `/users` | `http://userservice:8081` | `GET /users`, `GET /users/{id}` |
| `/orders` | `http://orderservice:8082` | `GET /orders`, `GET /orders/{id}` |
| `/healthz` | gateway | liveness (`status ok`) |
| `/metrics` | gateway | Prometheus scrape |

Unknown prefixes return `404`. A missing or unparseable upstream returns `502`. Breaker open returns `503`.

## Middleware order

Proxied traffic (`/` catch-all):

1. Request ID (`X-Request-ID` in or generated)
2. Request log
3. Panic recover → `500`
4. OpenTelemetry span
5. Prometheus request metrics
6. Auth — Bearer JWT **or** `X-API-Key`
7. Rate limit → `429`
8. Circuit breaker → reverse proxy

ServeMux also mounts `/metrics`, `/healthz`, and `/admin/*` (admin JWT on keys/health).

## Configuration

```yaml
listen: ":8088"
upstreams:
  users:
    url: "http://userservice:8081"
  orders:
    url: "http://orderservice:8082"
routes:
  - path: "/users"
    upstream: users
  - path: "/orders"
    upstream: orders
```

Hostnames assume Compose DNS (or Kubernetes Services of the same name).

| Variable | Default | Purpose |
|---|---|---|
| `JWT_SECRET` | required | HS256 signing key for user tokens |
| `REDIS_ADDR` | required | Redis `host:port` |
| `ADMIN_JWT_SECRET` | required | HS256 for admin tokens |
| `ADMIN_PASSWORD` | required | Admin login |
| `RATE_LIMIT_MAX` | `10` (`100` in Compose) | Max requests per window |
| `RATE_LIMIT_WINDOW_SEC` | `60` | Window length in seconds |
| `SAMPLE_API_KEY` | `demo-key` | Seeded into Redis at startup |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | optional | e.g. `jaeger:4318` |
| `FAIL_RATE` | (orders only) | Probability of 5xx; `0.8` in Compose |

## Observability

- **Prometheus** scrapes gateway `/metrics` (`gateway_http_requests_total`, duration histogram, `gateway_rate_limited_total`, circuit gauges/counters).
- **Grafana** is provisioned with that Prometheus datasource and a gateway dashboard.
- **Jaeger** receives OTLP HTTP traces from the gateway and mock services (`OTEL_EXPORTER_OTLP_ENDPOINT`).

## Kubernetes

Kustomize base in [`deploy/k8s/`](deploy/k8s/): Redis, users, orders, gateway (ConfigMap + Secret), nginx LoadBalancer.

Build images the manifests expect (`gateway:local`, `nginx-edge:local`, and the service images used in those YAMLs), then:

```powershell
kubectl apply -k deploy/k8s
```

Change `gateway-secrets` before any real cluster. Observability stack (Prometheus/Grafana/Jaeger) is Compose-only today.

## CI

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs on `main` and pull requests: `go test ./...`, golangci-lint, `docker build` of the gateway image.

## Layout

```
cmd/gateway/                 # gateway process
cmd/userservice/             # mock users API
cmd/orderservice/            # mock orders API (FAIL_RATE)
internal/auth/               # JWT + API key lookup
internal/ratelimit/          # Redis fixed-window limiter
internal/circuitbreaker/     # per-upstream closed/open/half-open
internal/proxy/              # route match + ReverseProxy
internal/middleware/         # request ID, recover, logger, OTel, metrics
internal/config/             # YAML load
internal/metrics/            # Prometheus registers
internal/tracing/            # OTLP setup
internal/admin/              # login, keys, health API
web/admin/                   # Vite + React admin SPA
nginx/                       # TLS + /api/ + static /admin/
configs/gateway.yaml
deploy/compose.yml
deploy/prometheus/           # scrape + Grafana provisioning
deploy/k8s/                  # kustomize
.github/workflows/ci.yml
Dockerfile                   # ARG CMD=gateway|userservice|orderservice
```

## Tests

```powershell
go test ./...
```
