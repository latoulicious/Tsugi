# Known Constraints

Operational facts and hidden contracts Tsugi must respect. Code is the source of
truth; flag drift here.

## Branch invariant (the point of Phase 1)

- **production deploys `main`, staging deploys `dev`.** Never wire production to
  `dev`. `deploy.sh` maps env → branch and refuses anything else.

## Single VPS

- One box runs both environments. Branches: `main`, `dev`. No staging branch —
  environment separation is sufficient.
- **Memory risk:** two fully isolated stacks (each its own postgres/redis/minio)
  roughly double RAM. The target's prod compose is already trimmed for ~4 GB, so
  two copies will not fit a 4 GB box as-is. Options when it bites: a
  shared-backing-services variant, a bigger box, or staging on demand only.

## Port map (published on 127.0.0.1)

| Service | Production | Staging |
|---|---|---|
| app edge (aegis) | 8080 | 8081 |
| minio S3 API | 9000 | 9100 |
| minio console | 9001 | 9101 |

Staging overrides shift these with the `!override` YAML tag (replaces the base
port list; plain merge would append and double-bind). Needs Docker Compose
v2.24.4+.

## Routing — Cloudflare Tunnel

- Two app hostnames: `api.<domain>` → 8080, `staging-api.<domain>` → 8081.
- Plus S3 hostnames (`s3.<domain>` → 9000, `staging-s3.<domain>` → 9100): the
  target's browser does presigned PUT straight to minio, and SigV4 binds the
  signature to that Host. Omit them and uploads fail.
- No open host ports; no host TLS. Tunnel terminates at Cloudflare.

## Target app (first target: LazyScan)

- Tsugi orchestrates `../LazyScan-Stack/docker-compose.prod.yml`; it does not
  fork it. Front service is `aegis` (not `web`).
- The api reads its own secrets from `LazyScan/api/.env` (env_file). Do not
  re-declare `JWT_SECRET`/`SUPERUSER_*` in Tsugi env-files — that blanks them.
- Base compose builds from contexts relative to its own dir, so the VPS checkout
  must be the full LazyScan-Stack repo at the env-specific path.

## Scaffold scope (2026-06-19)

- Config + runbook only. No live VPS changes, no real secrets, no `docker
  compose up` on the box. `.env*` and real cloudflared config are git-ignored.
- No Go code yet — Phases 2–6.
