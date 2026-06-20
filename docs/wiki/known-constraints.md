# Known Constraints

Operational facts and hidden contracts Tsugi must respect. Code is the source of
truth; flag drift here.

## Branch invariant (the point of Phase 1)

- **production deploys `main`, staging deploys `development`.** Never wire
  production to `development`. `deploy.sh` maps env → branch and refuses anything
  else.
- **Rollback exception (P6):** `deploy.sh --ref <sha>` does a *detached* checkout
  of a specific commit instead of `pull --ff-only`, so `tsugi release rollback`
  can land an older production release. The ref is still a prior point on `main`
  lineage — prod never points at `development`. Ref is charset-guarded
  (`[0-9a-fA-F]{7,40}`).

## Single VPS

- One box runs both environments. Branches: `main`, `dev`. No staging branch —
  environment separation is sufficient.
- **Memory risk:** two fully isolated stacks (each its own postgres/redis)
  roughly double RAM. The target's prod compose is already trimmed for ~4 GB, so
  two copies will not fit a 4 GB box as-is. Options when it bites: a
  shared-backing-services variant, a bigger box, or staging on demand only.

## Port map (published on 127.0.0.1)

| Service | Production | Staging |
|---|---|---|
| web (nginx edge) | 8081 | 8082 |

Staging overrides shift the port with the `!override` YAML tag (replaces the base
port list; plain merge would append and double-bind). Needs Docker Compose
v2.24.4+. On the box `8080` is dozzle and `8081` is the live prod stack, so
staging takes `8082`.

## Routing — Cloudflare Tunnel

- Two app hostnames on the shared `vps` tunnel: `lazyscan.my.id` → 8081,
  `staging.lazyscan.my.id` → 8082.
- No s3 hostname: uploads go browser → R2 directly (presigned PUT to R2's own
  public domain), so the tunnel only carries the nginx `web` edge.
- No open host ports; no host TLS. Tunnel terminates at Cloudflare.

## Target app (first target: LazyScan)

- Tsugi orchestrates `../LazyScan/docker-compose.yml`; it does not fork it. The
  front service is `web` (nginx: SPA + `/api` proxy), the sole published edge.
- The compose has no `env_file:`; it interpolates one root `.env` in the checkout
  (`JWT_SECRET`, `STORAGE_*` R2, `SMTP_*`, `DEFAULT_SUPERUSER_*`, …). Tsugi passes
  `--project-directory <checkout>` so that `.env` is used — it injects no env-file.
- Base compose builds from contexts relative to its own dir, so the VPS checkout
  must be the full LazyScan repo at the env-specific path.

## Tsugi's own storage (P5+)

- Tsugi's `releases`/`deployments` tables live on the **existing shared
  Postgres** the box already runs (`postgres:16-alpine`, same instance as
  LazyScan), in its own `tsugi` database. No dedicated container.
- **One instance, both environments.** Unlike the target's per-stack isolation,
  Tsugi keeps a single unified history — the `deployments.environment` column
  (`staging`/`production`) is the only distinction. Deployment history must be
  queryable across both envs, so it cannot be split per stack.
- Connection via `TSUGI_DATABASE_URL` (P6: required by `tsugi migrate`/`release`,
  not by `serve`). Migrations in `migrations/` are embedded and applied with
  `tsugi migrate up` (tracked in `schema_migrations`); the runner is minimal — no
  dirty-state recovery, no down-to-version. `down` rolls back the last step only.

## Scaffold scope (2026-06-19)

- Config + runbook only. No live VPS changes, no real secrets, no `docker
  compose up` on the box. `.env*` and real cloudflared config are git-ignored.
- No Go code yet — Phases 2–6.
