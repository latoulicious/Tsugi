# Module Docs

Per-package documentation for the Tsugi Go service. Package/symbol docs live
here instead of as in-code godoc comments. Code is the source of truth — flag
drift.

## Packages

| Package | Path | Doc |
|---|---|---|
| main | `cmd/tsugi` | entrypoint: `serve` / `migrate` / `release` dispatch + wiring |
| version | `internal/version` | [version.md](version.md) — build identity for `GET /version` |
| config | `internal/config` | [config.md](config.md) — env-driven runtime config |
| server | `internal/server` | [server.md](server.md) — HTTP routes `/version`, `/healthz` |
| release | `internal/release` | [release.md](release.md) — P3 release entity + lifecycle state machine |
| changelog | `internal/changelog` | [changelog.md](changelog.md) — P4 conventional-commit changelog generation |
| deployment | `internal/deployment` | [deployment.md](deployment.md) — P5 deployment-history entity + repository port |
| postgres | `internal/postgres` | [postgres.md](postgres.md) — P5 pgx adapter + P6 tx/migrations |
| git | `internal/git` | [git.md](git.md) — P6 git exec wrapper (changelog input) |
| deploy | `internal/deploy` | [deploy.md](deploy.md) — P6 adapter shelling out to `deploy.sh` |
| deployflow | `internal/deployflow` | [deployflow.md](deployflow.md) — P5.4 shared production-deploy use case (cli + agent) |
| cli | `internal/cli` | [cli.md](cli.md) — P6 `release` CLI use-cases |

## Layering

Flat `internal/*` layout — parity with the LazyScan-Stack Go services
(Aegis/Herald/Kiln), which are also flat. Domain packages (`release`,
`deployment`) define their own `Repository` port; `postgres` is the
infrastructure adapter implementing both. P6 adds the application layer
(`cli`, the use-cases) and two more infra adapters (`git`, `deploy`) — kept as
flat peers rather than a `domain/application/infrastructure` subtree, matching
the rest of the stack.
