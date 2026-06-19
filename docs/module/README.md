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
| deployment | `internal/deployment` | [deployment.md](deployment.md) — P5 deployment-history entity + repository port |
| postgres | `internal/postgres` | [postgres.md](postgres.md) — P5 pgx adapter for the release/deployment ports |

## Layering

Flat `internal/{config,server,version,release,changelog,deployment,postgres}`
layout — parity with the LazyScan-Stack Go services (Aegis/Herald/Kiln), which
are also flat. Domain packages (`release`, `deployment`) define their own
`Repository` port; the single `postgres` package is the infrastructure adapter
implementing both. No `application`/`interfaces` split yet — that arrives with
the P6 CLI use-cases.
