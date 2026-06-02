# Kerbecs

<img align="right" width="159px" src="assets/kerbecs.png" alt="kerbecs-logo">

[![Build Status](https://github.com/BK1031/Kerbecs/actions/workflows/test.yml/badge.svg)](https://github.com/BK1031/Kerbecs/actions/workflows/test.yml)
[![GoDoc](https://pkg.go.dev/badge/github.com/bk1031/kerbecs?status.svg)](https://pkg.go.dev/github.com/bk1031/kerbecs?tab=doc)
[![Docker Pulls](https://img.shields.io/docker/pulls/bk1031/kerbecs?style=flat-square)](https://hub.docker.com/repository/docker/bk1031/kerbecs)
[![Release](https://img.shields.io/github/release/bk1031/kerbecs.svg?style=flat-square)](https://github.com/bk1031/kerbecs/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Kerbecs is a standalone HTTP API gateway written in [Go](https://go.dev/). It
routes incoming requests to upstream services based on a YAML config,
optionally wraps responses in a consistent envelope, balances across multiple
instances per upstream, and is built around a pluggable `Provider` abstraction
so additional routing sources (service registries, orchestrators) can plug in
without touching the rest of the system.

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
    instances:
      - http://users-1:8080
      - http://users-2:8080
    load_balancer: round_robin
    timeouts: { dial: 2s, headers: 5s, idle: 50s }

routes:
  - name: users-api
    match: { path: /users/*, methods: [GET, POST, PUT, DELETE] }
    upstream: users-service
    envelope: default
    timeouts:
      overall: 10s
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
3. **Proxy handler** resolves the matched route, applies any path rewrite,
   picks an upstream instance via the configured load balancer, and
   reverse-proxies using a per-upstream `http.Transport` with a shared
   connection pool and dial / headers / idle timeouts.

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

  When rewriting, the response's `Content-Length` is recomputed and any
  upstream `Transfer-Encoding: chunked` is cleared so the response is
  unambiguous.

- **`envelope: passthrough`** streams the response unchanged.

Envelope routes automatically fall through to passthrough for content that
must stream: WebSocket upgrades (HTTP 101), `text/event-stream` (SSE),
`application/grpc`, and common binary MIMEs (octet-stream, zip, pdf, csv).

## Load balancing

Each upstream lists one or more `instances` and picks via a strategy:

| `load_balancer` | Behavior |
|---|---|
| `round_robin` (default) | Atomic counter rotates through instances in declared order |
| `random` | Uniform random pick per request |

Both are concurrency-safe and lock-free in the hot path. Least-connections,
weighted, and consistent-hash strategies are not yet implemented.

There are no active health checks today. If an instance is unreachable, the
load balancer keeps it in rotation and `~1/N` of requests will fail with `502`
until you remove it from config.

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

## Timeouts

Four timeouts bound different phases of a proxied request:

| Timeout | What it bounds | Default |
|---|---|---|
| `dial` | TCP/TLS handshake to the upstream | `5s` |
| `headers` | Time from request sent to first byte of response headers (TTFB) | `30s` |
| `idle` | How long an idle connection sits in the pool before being closed | `90s` |
| `overall` | Total per-request budget (request → headers → body complete) | `0` (no deadline) |

Configure at three levels, with later overriding earlier:

```yaml
gateway:
  timeouts: { dial: 5s, headers: 30s, idle: 90s, overall: 0 }   # global default

upstreams:
  api-service:
    timeouts: { idle: 50s }                                      # ALB-friendly override

routes:
  - name: search
    upstream: api-service
    timeouts: { overall: 60s }                                   # cold-cache search
```

`dial`, `headers`, and `idle` are connection-pool concerns and live on the
per-upstream `Transport`, so per-route override is intentionally not
supported — define a separate upstream pointing at the same instances if you
need that. `overall` is per-request and overridable per-route.

WebSocket upgrades (`Upgrade: websocket`) automatically bypass `overall`.
SSE and large-download routes should set `overall: 0` explicitly so the
stream isn't cut short.

`overall` deadline exceeded returns `504 Gateway Timeout`, enveloped if the
matched route requested an envelope.

`idle` matters most when something with its own keep-alive timeout sits
between you and the upstream. AWS ALB closes idle connections at `60s` by
default; a Kerbecs `idle` longer than that produces sporadic
`connection reset` errors during low-traffic windows. Keep it strictly under
the front-end's idle timeout.

## CORS

Configurable per listener under `listeners.gateway.cors` and
`listeners.admin.cors`. Off by default on both. Example:

```yaml
listeners:
  gateway:
    port: "10310"
    cors:
      enabled: true
      allowed_origins: [https://app.example.com]
      allow_credentials: true
      max_age: 12h
```

`allow_all_origins: true` works but emits the wildcard echoing pattern that
most security guidance flags when combined with `allow_credentials: true`.
Prefer an explicit allowlist.

CORS is currently listener-scoped (one policy applies to all routes on that
listener). Per-route CORS will arrive once the middleware runtime ships.

## Lifecycle

`SIGINT` / `SIGTERM` trigger a graceful drain. In-flight requests complete up
to a 30-second deadline before the process exits. During the drain the
listener stops accepting new connections but existing responses are served to
completion.

## Admin server

A separate HTTP listener (default `:10300`) exposes Kerbecs's own endpoints.
Basic auth is enforced for everything except `/admin-gw/ping`, and credentials
are compared in constant time. Configure via `listeners.admin` in the YAML.
The only built-in endpoint today is the health ping; richer admin actions
(drain, route inspection, log-level toggle) are future work.

## Environment variables

The YAML config is authoritative. Env vars are useful only when referenced
from YAML via `${VAR}` or `${VAR:default}` substitution — except for two:

| Variable            | Read directly by Kerbecs?                              |
|---------------------|--------------------------------------------------------|
| `KERBECS_CONFIG`    | Yes — path to the config file (default `kerbecs.yaml`) |
| `ENV`               | Yes — `PROD` selects the JSON production logger at startup |

All other "well-known" names (`PORT`, `ADMIN_PORT`, `KERBECS_USER`,
`KERBECS_PASSWORD`, etc.) are conventions used by the example config's
`${VAR:default}` expansions. They have no effect unless the YAML you load
actually references them.

## Configuration reloading

With `providers.static.watch: true`, Kerbecs reloads the config file on change
and atomically swaps the live routing state — no restart, no dropped
connections. If the new file fails to parse or build, the previous config stays
in place and the error is logged, so a bad edit never takes the gateway down.
When `watch` is `false` (the default), the config is read once at startup.

Two reload mechanisms are available via `watch_mode`:

| `watch_mode`     | How it detects changes                          | When to use it |
|------------------|-------------------------------------------------|----------------|
| `file` (default) | Filesystem events (`fsnotify`, inotify/kqueue). | Local files and most volume mounts. Lowest latency. |
| `poll`           | Stats the file every `watch_interval`.          | When file events aren't delivered for the config path — some network/overlay mounts and container volume drivers never fire them. |

```yaml
providers:
  static:
    watch: true
    watch_mode: file        # "file" (default) or "poll"

# or, to poll instead of relying on filesystem events:
providers:
  static:
    watch: true
    watch_mode: poll
    watch_interval: 5s      # poll period; defaults to 5s
```

Both modes are Kubernetes ConfigMap–aware: a ConfigMap update swaps the mounted
`..data` symlink, which `file` mode observes as a directory event and `poll`
mode detects as a change in the resolved symlink target. (This requires a
whole-directory ConfigMap mount, not a `subPath` mount — `subPath` volumes
don't receive updates.) An unknown `watch_mode` logs a warning and falls back
to `file`.

## Current capabilities

- HTTP/1.1 ingress; HTTP/2 upstream negotiation via `ForceAttemptHTTP2`
- WebSocket upgrade passthrough
- SSE / gRPC / binary content auto-passthrough on envelope routes
- Streaming request bodies (no gateway-side buffering)
- Multi-instance upstreams with round-robin or random load balancing
- Per-upstream connection pool with `dial` / `headers` / `idle` timeouts
- Per-request `overall` timeout with `504` on deadline, WebSocket bypass
- Path rewrites (`strip_prefix`, `replace_prefix`)
- First-match routing with exact / prefix / regex path patterns
- Per-route byte caps for requests and responses
- Per-listener CORS (allowlist or wildcard, off by default)
- Hot config reload via file events or polling, with atomic state swap
- Graceful shutdown with in-flight drain
- Constant-time admin auth

## Not yet supported

- **Middleware runtime.** `routes[].middlewares: [...]` parses but is a no-op.
  JWT, rate limiting, and per-route policy are blocked on this.
- **Active health checks.** `health_check` config is parsed but unused; dead
  instances stay in load-balancer rotation.
- **Prometheus metrics and OpenTelemetry tracing.**
- **TLS termination** on the listener. Run behind a TLS-terminating LB.
- **Non-static providers** (service registry, Docker labels, Kubernetes
  ingress).

## Contributing

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".
Don't forget to give the project a star! Thanks again!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b gh-username/my-amazing-feature`)
3. Commit your Changes (`git commit -m 'Add my amazing feature'`)
4. Push to the Branch (`git push origin gh-username/my-amazing-feature`)
5. Open a Pull Request

## License

Distributed under the MIT License. See `LICENSE.txt` for more information.
