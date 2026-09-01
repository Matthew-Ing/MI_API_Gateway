# MI API Gateway

A Go reverse proxy in front of two mock services. Auth (JWT or API key) and Redis rate limiting sit in the request path; routing is YAML, not a framework mux per backend.

Built with Go `net/http` and `httputil.ReverseProxy` — no chi, gin, or echo.

## Architecture

```mermaid
flowchart LR
  client[Client] -->|":8089"| gateway[Go gateway]
  gateway --> redis[Redis]
  gateway --> users[User service]
  gateway --> orders[Order service]
```

Request path: **auth** → **rate limit** → **prefix match on** `configs/gateway.yaml` → **reverse proxy**. Redis stores hashed API keys and per-identity counters. `/healthz` is unauthenticated and skips that chain.

## Features

- YAML route table: prefix match (`/users` covers `/users` and `/users/1`)
- JWT (`Authorization: Bearer`) or API key (`X-API-Key`)
- API keys stored as SHA-256 hashes in Redis (`apikey:<hex>`)
- Fixed-window rate limit in Redis (`429` + `Retry-After`)
- Middleware: request ID, access log, panic recover
- Docker Compose: gateway, Redis, user service, order service

## Quick start

From the repo root:

```powershell
docker compose -f deploy/compose.yml up --build
```

Gateway is on `http://localhost:8089` (container listens on `:8088`).

```powershell
curl.exe http://localhost:8089/healthz
# status ok
```

Unauthenticated proxied routes are rejected:

```powershell
curl.exe -i http://localhost:8089/users
# HTTP/1.1 401 Unauthorized
```

### API key

Store the SHA-256 of the raw key, then send the raw key on the request.

```powershell
docker compose -f deploy/compose.yml exec redis redis-cli SET apikey:c48a01f49fd0f2cc404bc3cbbc80e91457a3d41bb429a695243de4c61794155c 1

curl.exe -H "X-API-Key: demo-key" http://localhost:8089/users
curl.exe -H "X-API-Key: demo-key" http://localhost:8089/users/1
curl.exe -H "X-API-Key: demo-key" http://localhost:8089/orders
```

Hash for a different key:

```powershell
python -c "import hashlib; print(hashlib.sha256(b'your-key').hexdigest())"
```

### JWT

Compose sets `JWT_SECRET=secret`. Tokens are HS256 with a `userID` claim.

```powershell
curl.exe -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySUQiOiJhbGljZSJ9.Qv5mGwg-1oJrJCGaUhrJ9mleOFdEFHrGscn5-5p7dnk" http://localhost:8089/users
```

Mint your own with the same secret (HS256, payload `{"userID":"..."}`).

### Rate limit

Default is **10 requests per 60 seconds** per identity (API key hash, Bearer token hash, or client address). The 11th request in the window:

```text
HTTP/1.1 429 Too Many Requests
Retry-After: 60
```

```powershell
1..12 | ForEach-Object {
  curl.exe -s -o NUL -w "%{http_code}`n" -H "X-API-Key: demo-key" http://localhost:8089/users
}
```

## Routes

From [`configs/gateway.yaml`](configs/gateway.yaml):

| Path prefix | Upstream | Service |
|---|---|---|
| `/users` | `http://userservice:8081` | `GET /users`, `GET /users/{id}` |
| `/orders` | `http://orderservice:8082` | `GET /orders`, `GET /orders/{id}` |
| `/healthz` | gateway | liveness (`status ok`) |

Unknown prefixes return `404`. A route whose upstream is missing or unparseable returns `502`.

## Middleware order

1. Request ID (`X-Request-ID` in or generated)
2. Request log
3. Panic recover → `500`
4. Auth — Bearer JWT **or** `X-API-Key`
5. Rate limit → `429`
6. Reverse proxy

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

Hostnames assume Compose DNS. The gateway process requires `REDIS_ADDR` and `JWT_SECRET` at startup.

| Variable | Default | Purpose |
|---|---|---|
| `JWT_SECRET` | required | HS256 signing key |
| `REDIS_ADDR` | required | Redis `host:port` |
| `RATE_LIMIT_MAX` | `10` | Max requests per window |
| `RATE_LIMIT_WINDOW_SEC` | `60` | Window length in seconds |

## Layout

```
cmd/gateway/           # gateway process
cmd/userservice/       # mock users API
cmd/orderservice/      # mock orders API
internal/auth/         # JWT validate + API key lookup
internal/ratelimit/    # Redis fixed-window limiter
internal/proxy/        # route match + ReverseProxy
internal/middleware/   # request ID, recover, logger, chain
internal/config/       # YAML load
configs/gateway.yaml
deploy/compose.yml
Dockerfile             # ARG CMD=gateway|userservice|orderservice
```

## Tests

```powershell
go test ./...
```

## Roadmap

Circuit breaker, OpenTelemetry, Prometheus/Grafana, nginx at the edge, and a small admin UI for keys are planned next — not in this tree yet.
