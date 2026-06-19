# internal/server

Wires Tsugi's HTTP routes onto an `http.Server`.

## Symbols

- `New(cfg *config.Config, logger *slog.Logger) *http.Server` — builds the
  server with a `ServeMux` and timeouts (`ReadHeaderTimeout`, `IdleTimeout`).

## Routes

| Method + Path | Response |
|---|---|
| `GET /version` | `version.Info` JSON `{version, commit, deployed_at}` |
| `GET /healthz` | `{"status":"ok","service":"tsugi"}` — liveness, compose healthcheck parity |

Method-qualified patterns (Go 1.22+) — a wrong method returns `405`, an unknown
path `404`.
