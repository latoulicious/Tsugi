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
  api.<domain>         -> 127.0.0.1:8080  production stack (branch main)
  s3.<domain>          -> 127.0.0.1:9000  production minio (presigned PUT)
  staging-api.<domain> -> 127.0.0.1:8081  staging stack    (branch dev)
  staging-s3.<domain>  -> 127.0.0.1:9100  staging minio

production  = docker compose -p <target>-prod    (checkout @ main)
staging     = docker compose -p <target>-staging (checkout @ dev)
```

Two isolated compose project stacks, distinguished by project name, published
port offset, env-file, and checked-out branch. Isolation is total (each stack
has its own postgres/redis/minio) — see the memory note in
`known-constraints.md`.

## Deploy Mechanism (Phase 1)

Tsugi adds four per-environment levers on top of the target's base compose:

| Lever | Production | Staging |
|---|---|---|
| compose project (`-p`) | `lazyscan-prod` | `lazyscan-staging` |
| override file | `docker-compose.prod.override.yml` | `docker-compose.staging.override.yml` |
| env-file | `.env.prod` | `.env.staging` |
| checkout branch | `main` | `dev` |

Override files only carry the port deltas (via the `!override` YAML tag, which
replaces the base list instead of appending). Everything else comes from the
target's own compose.

## Packages

The Go service lands with Phase 2 (stdlib only — pgx arrives with the P5
tables). Per-package docs live in `docs/module/` (not in-code godoc).

```txt
cmd/tsugi          entrypoint: load config, start server, graceful shutdown
internal/version   build identity (Version/Commit/Date via ldflags) for /version
internal/config    env-driven runtime config (TSUGI_ADDR)
internal/server    HTTP routes: GET /version, GET /healthz
internal/release   P3 release entity + lifecycle state machine (pure domain)
```

Flat layout, parity with the LazyScan-Stack Go services. P3 adds the `release`
domain package as a peer; the layered DDD split (application / infrastructure /
interfaces) is deferred to P5 persistence and P6 CLI, when adapters/use-cases
need it. No repository port yet (lands in P5 with the pgx impl).

## Phases (P1–P6, from `infra-plan.md`)

| Phase | What | Status |
|---|---|---|
| P1 | Environment separation: staging/prod containers, domains, deploy targets | **in progress** — scaffold 2026-06-19 |
| P2 | Version visibility: `GET /version` (version, commit, deployed_at) | **done** — scaffold 2026-06-19 |
| P3 | Release management: release metadata + state machine | **done** — scaffold 2026-06-19 |
| P4 | Changelog generation from `git log` (conventional commits, no AI) | planned |
| P5 | Deployment tracking: `releases` + `deployments` tables | planned |
| P6 | Promotion & rollback: `release` CLI | planned |

## Current Behavior (P1 scaffold)

No running service. The repo carries the deploy layer and project memory only.
`deploy/bin/deploy.sh --target lazyscan --env <prod|staging>` resolves the
branch, project, override, and env-file, then runs `docker compose ... up -d`
against the target checkout. The Go `release` CLI (Phase 6) supersedes this
script. Nothing here touches a live VPS — operators run it on the box.
