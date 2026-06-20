# Architecture

Tsugi is a release-promotion and deployment-orchestration tool for personal
infrastructure on a single VPS. It moves a release forward — staging → validate
→ promote → production — with version visibility, changelogs, and rollback.

Tsugi **orchestrates** target apps; it does not own or fork their compose. The
first target is LazyScan. Phases 2–6 add a Go service (`GET /version`) and a
`release` CLI backed by Postgres.

## Target Topology (single VPS)

```txt
Cloudflare Tunnel
  lazyscan.my.id          -> 127.0.0.1:8081  production stack (branch main)
  staging.lazyscan.my.id  -> 127.0.0.1:8082  staging stack    (branch development)

production  = docker compose -p lazyscan         (checkout @ main)
staging     = docker compose -p lazyscan-staging (checkout @ development)
```

One nginx `web` edge per stack (serves the SPA and proxies `/api`, same origin).
Uploads go browser → R2 directly, so there is no s3 hostname. Two isolated
compose project stacks, distinguished by project name, published port offset,
the checkout's own `.env`, and checked-out branch. Each stack runs its own
postgres/redis — see the memory note in `known-constraints.md`.

## Deploy Mechanism (Phase 1)

Tsugi adds four per-environment levers on top of the target's base compose:

| Lever | Production | Staging |
|---|---|---|
| compose project (`-p`) | `lazyscan` | `lazyscan-staging` |
| override file | `docker-compose.prod.override.yml` | `docker-compose.staging.override.yml` |
| published port | `127.0.0.1:8081` | `127.0.0.1:8082` |
| interpolation | checkout's `.env` | checkout's `.env` |
| checkout branch | `main` | `development` |

Override files only carry the port deltas (via the `!override` YAML tag, which
replaces the base list instead of appending). Everything else comes from the
target's own compose.

## Packages

The Go service lands with Phase 2 (stdlib only until P5, which adds pgx for the
release/deployment tables). Per-package docs live in `docs/module/` (not in-code
godoc).

```txt
cmd/tsugi          entrypoint: serve / migrate / release dispatch + wiring
internal/version   build identity (Version/Commit/Date via ldflags) for /version
internal/config    env-driven runtime config (TSUGI_ADDR, TSUGI_DATABASE_URL, ...)
internal/server    HTTP routes: GET /version, GET /healthz
internal/release   P3 release entity + lifecycle state machine (pure domain)
internal/changelog P4 conventional-commit changelog generation (pure, no git)
internal/deployment P5 deployment-history entity + repository port (pure domain)
internal/postgres  P5 pgx adapter + P6 WithTx/Connect + migration runner
internal/git       P6 git exec wrapper (HeadSHA, Subjects) — changelog input
internal/deploy    P6 adapter shelling out to deploy/bin/deploy.sh
internal/cli       P6 release CLI use-cases (create/list/show/promote/rollback)
```

Flat layout, parity with the sibling LazyScan Go services. Domain packages
(`release`, `deployment`) each define a `Repository` port; `postgres`/`git`/
`deploy` are the infrastructure adapters; `cli` is the application layer (the
use-cases). Kept as flat peers, not a `domain/application/infrastructure`
subtree — matching the rest of the stack. P5 is the first external dependency
(`github.com/jackc/pgx/v5`).

## Phases (P1–P6, from `infra-plan.md`)

| Phase | What | Status |
|---|---|---|
| P1 | Environment separation: staging/prod containers, domains, deploy targets | **in progress** — scaffold 2026-06-19 |
| P2 | Version visibility: `GET /version` (version, commit, deployed_at) | **done** — scaffold 2026-06-19 |
| P3 | Release management: release metadata + state machine | **done** — scaffold 2026-06-19 |
| P4 | Changelog generation from `git log` (conventional commits, no AI) | **done** — scaffold 2026-06-19 |
| P5 | Deployment tracking: `releases` + `deployments` tables | **done** — scaffold 2026-06-19 |
| P6 | Promotion & rollback: `release` CLI (create/list/show/promote/rollback) | **done** — scaffold 2026-06-19 |
| P7 | Target topology correction: re-point deploy scaffold + wiki from legacy `LazyScan-Stack` to the live `../LazyScan` monorepo | **in progress** — repo sweep 2026-06-20; VPS staging stand-up pending |

## Current Behavior (P1 scaffold)

No running service. The repo carries the deploy layer and project memory only.
`deploy/bin/deploy.sh --target lazyscan --env <prod|staging>` resolves the
branch, project, and port override, then runs `docker compose ... up -d`
against the target checkout (interpolation from the checkout's own `.env`). The Go `release` CLI (Phase 6) supersedes this
script. Nothing here touches a live VPS — operators run it on the box.
