# Kerbecs

<img align="right" width="159px" src="assets/kerbecs.png" alt="kerbecs-logo">

[![Build Status](https://github.com/BK1031/Kerbecs/actions/workflows/test.yml/badge.svg)](https://github.com/BK1031/Kerbecs/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/BK1031/Kerbecs/graph/badge.svg?token=R4NMABYGOZ)](https://codecov.io/gh/BK1031/Kerbecs)
[![GoDoc](https://pkg.go.dev/badge/github.com/bk1031/kerbecs?status.svg)](https://pkg.go.dev/github.com/bk1031/kerbecs?tab=doc)
[![Docker Pulls](https://img.shields.io/docker/pulls/bk1031/kerbecs?style=flat-square)](https://hub.docker.com/repository/docker/bk1031/kerbecs)
[![Release](https://img.shields.io/github/release/bk1031/kerbecs.svg?style=flat-square)](https://github.com/bk1031/kerbecs/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Kerbecs is a standalone HTTP API gateway written in [Go](https://go.dev/). It
routes incoming requests to upstream services based on a YAML config,
optionally wraps responses in a consistent envelope, and is built around a
pluggable `Provider` abstraction so additional routing sources (service
registries, orchestrators) can plug in without touching the rest of the
system.

## Quick start

Kerbecs reads its config from `kerbecs.yaml` in the working directory, or from
`$KERBECS_CONFIG`:

```bash
docker run -d -p 10310:10310 \
    -v "$(pwd)/kerbecs.yaml:/etc/kerbecs/kerbecs.yaml" \
    -e KERBECS_CONFIG=/etc/kerbecs/kerbecs.yaml \
    bk1031/kerbecs:latest
```

Minimal config:

```yaml
gateway:
  name: kerbecs-edge
  version: 3.0.0

listeners:
  gateway: { port: "10310" }
  admin:
    port: "10300"
    auth: { type: basic, username: admin, password: admin }

providers:
  static: { watch: false }

upstreams:
  users-service:
    name: users
    version: 1.0.0
    instances: [http://users:8080]
    timeouts: { dial: 2s, response_header: 5s, idle: 90s }

routes:
  - name: users-api
    match: { path: /users/*, methods: [GET, POST, PUT, DELETE] }
    upstream: users-service
    envelope: default
```

See [`examples/kerbecs.yaml`](examples/kerbecs.yaml) for a fuller reference.

## How it works

Kerbecs has three layers:

1. **Providers** supply routes. The built-in static provider reads from the
   config file. Additional providers (service registry, Docker labels,
   Kubernetes ingress) plug in behind the same interface.
2. **Router** compiles routes into a first-match table. Path patterns support
   exact (`/foo` or `exact:/foo`), prefix (`/foo/*`), and regex
   (`regex:^/x/\d+$`) forms, plus method and host filters.
3. **Proxy handler** resolves the matched route, applies any path rewrite, and
   reverse-proxies to the upstream using a per-upstream `http.Transport` with
   a shared connection pool and dial / response-header / idle timeouts.

## Envelope

Each route declares how it treats upstream responses:

- **`envelope: default`** buffers the response and wraps it:

  ```json
  {
    "status": "SUCCESS",
    "ping": "3ms",
    "gateway": "kerbecs-edge:v3.0.0",
    "service": "users:v1.0.0",
    "timestamp": "Fri Apr 24 14:19:50 PDT 2026",
    "data": { /* upstream body */ }
  }
  ```

- **`envelope: passthrough`** streams the response unchanged.

Envelope routes automatically fall through to passthrough for content that
must stream: WebSocket upgrades (HTTP 101), `text/event-stream` (SSE),
`application/grpc`, and common binary MIMEs (octet-stream, zip, pdf, csv).

## Request and response caps

Every route carries byte caps (default 100 MiB each, overridable per route):

```yaml
gateway:
  limits:
    max_request_bytes: 100MB
    max_response_bytes: 100MB

routes:
  - name: upload
    match: { path: /upload }
    upstream: storage
    limits:
      max_request_bytes: 500MB    # per-route override
```

- Oversized requests return `413 Payload Too Large`. Enforcement uses
  `http.MaxBytesReader`, so a lying `Content-Length` header doesn't bypass the
  cap — the read is terminated at the byte boundary.
- Oversized responses on envelope routes return `502 Bad Gateway` with an
  enveloped error. Passthrough routes stream and are not capped.

Sizes accept `100MB`, `500KB`, `1GiB`, or raw byte counts. All multipliers
are binary (1024-based).

## Lifecycle

`SIGINT` / `SIGTERM` trigger a graceful drain. In-flight requests complete up
to a 30-second deadline before the process exits. During the drain the
listener stops accepting new connections but existing responses are served to
completion.

## Admin server

A separate HTTP listener (default `:10300`) exposes Kerbecs's own endpoints.
Basic auth is enforced for everything except `/admin-gw/ping`, and credentials
are compared in constant time. Configure via `listeners.admin` in the YAML.

## Environment variables

These override their config-file counterparts or feed the `${VAR}` /
`${VAR:default}` substitution in the YAML:

| Variable            | Purpose                                          |
|---------------------|--------------------------------------------------|
| `KERBECS_CONFIG`    | Path to the config file (default `kerbecs.yaml`) |
| `PORT`              | Gateway listener port                            |
| `ADMIN_PORT`        | Admin listener port                              |
| `ENV`               | `PROD` switches gin to release mode              |
| `KERBECS_USER`      | Admin basic-auth username                        |
| `KERBECS_PASSWORD`  | Admin basic-auth password                        |
| `USE_CORS`          | `true` enables global wildcard CORS (temporary)  |

## Current capabilities

- HTTP/1.1 ingress; HTTP/2 upstream negotiation via `ForceAttemptHTTP2`
- WebSocket upgrade passthrough
- SSE / gRPC / binary content auto-passthrough on envelope routes
- Streaming request bodies (no gateway-side buffering)
- Per-upstream connection pool with configurable timeouts
- Path rewrites (`strip_prefix`, `replace_prefix`)
- First-match routing with exact / prefix / regex path patterns
- Per-route byte caps for requests and responses
- Graceful shutdown with in-flight drain

## Not yet supported

- Middleware runtime — config parses `middlewares: [...]` on routes but the
  runtime is a no-op. JWT, rate limiting, and CORS allowlists land in the
  middleware phase.
- Multi-instance upstreams and load balancing — only the first instance is
  used today.
- Active health checks.
- Hot reload on config change.
- Prometheus metrics and OpenTelemetry tracing.
- Non-static providers (service registry / Docker / Kubernetes).

## License

MIT.
