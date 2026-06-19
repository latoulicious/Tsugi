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
| release | `internal/release` | [release.md](release.md) — P3 release entity + lifecycle state machine |
| changelog | `internal/changelog` | [changelog.md](changelog.md) — P4 conventional-commit changelog generation |

## Layering

Flat `internal/{config,server,version,release,changelog}` layout — parity with the
LazyScan-Stack Go services (Aegis/Herald/Kiln), which are also flat. P3 adds the
first domain package (`release`) as a peer, not a `domain/` subtree: the layered
split (`application` / `infrastructure` / `interfaces`) is deferred until P5
persistence and P6 CLI add adapters and use-cases that need it.
