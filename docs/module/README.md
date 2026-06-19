# Module Docs

Per-package documentation for the Tsugi Go service. Package/symbol docs live
here instead of as in-code godoc comments. Code is the source of truth — flag
drift.

## Packages

| Package | Path | Doc |
|---|---|---|
| main | `cmd/tsugi` | entrypoint: load config, start server, graceful shutdown |
| version | `internal/version` | [version.md](version.md) — build identity for `GET /version` |
| config | `internal/config` | [config.md](config.md) — env-driven runtime config |
| server | `internal/server` | [server.md](server.md) — HTTP routes `/version`, `/healthz` |

## Layering

Phase 2 is a flat `internal/{config,server,version}` layout — parity with the
LazyScan-Stack Go services (Aegis/Herald/Kiln), which are also flat. Full DDD
layering (`domain` / `application` / `infrastructure` / `interfaces`) is
deferred to P3+ when the `releases` and `deployments` models arrive; there is no
domain entity to model yet.
