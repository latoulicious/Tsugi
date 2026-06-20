# Tsugi Infra Plan

Project-local copy of the original `PLAN.md` (Release Promotion Platform),
implemented as **Tsugi**. Keep future per-phase status here; the original plan is
kept verbatim below for history.

Type: Go release-promotion + deployment-orchestration service + CLI.

Status: **P6 scaffold done** — `release` CLI (create/list/show/promote/rollback)
+ `migrate` runner + git/deploy adapters + `WithTx` (2026-06-19). P5 `deployment`
entity + `postgres` pgx adapter + migrations (2026-06-19). P4 `changelog`
conventional-commit generator (2026-06-19). P3 `release` domain entity + state
machine (2026-06-19). P2 `GET /version` service (2026-06-19). P1 environment-
separation scaffold (2026-06-19).

## 2026-06-19 Update — Phase 1 scaffold (environment separation)

- Repo bootstrapped from empty: `AGENTS.md`, `README.md`, `.gitignore`,
  `docs/wiki/` (docs convention follows `../LazyScan`), and the generic
  `deploy/` layer.
- **Decisions** (confirmed): runtime is a Go service + CLI (Phases 2–6, not
  built yet); routing is Cloudflare Tunnel with two hostnames; the deploy layer
  is generic with **LazyScan as the first target**; this phase is config +
  runbook only — no live VPS, no real secrets.
- **Mechanism**: per-env compose project name (`<target>-prod` / `-staging`),
  per-env port override (`!override`), per-env `--env-file`, per-env branch
  checkout (`main` / `dev`). Tsugi orchestrates the target's own compose
  (`../LazyScan-Stack/docker-compose.prod.yml`), it does not fork it.
- **Deliverables**: `deploy/targets/lazyscan/` (prod + staging port overrides,
  `.env.*.example`, `cloudflared/config.yml.example`, `target.env`),
  `deploy/bin/deploy.sh` (interim deployer), `deploy/README.md`.
- **Known risk** (logged in `known-constraints.md`): two full isolated stacks on
  one small VPS roughly doubles memory — the prod compose is already trimmed for
  4 GB. A lighter shared-backing-services variant is possible later.
- Go skeleton (`go.mod`, `cmd/tsugi`, `internal/`) deliberately deferred to P2.

## 2026-06-19 Update — Phase 2 scaffold (version visibility)

- First Go code. Module `github.com/latoulicious/Tsugi`, go 1.26, **stdlib
  only** (pgx deferred to P5). `cmd/tsugi` + `internal/{version,config,server}`.
- `GET /version` → `{version, commit, deployed_at}`; `GET /healthz` for compose
  healthcheck parity. `net/http` + `slog` + graceful shutdown — mirrors Aegis.
- **Build identity** stamped at link time (`-X` ldflags) from git via the
  `Makefile`; `Dockerfile` takes the same as `--build-arg`. `deployed_at` =
  build time (rebuild-on-deploy ⇒ build ≈ deploy). Local `go build` falls back
  to the embedded VCS stamp (`debug.ReadBuildInfo`).
- Package docs in `docs/module/` (project preference: no in-code godoc).
- **Deferred**: Tsugi's own compose / VPS deploy wiring (overlaps P5/P6); the
  service is build- and run-validated locally only.

## 2026-06-19 Update — Phase 3 scaffold (release management)

- First domain code. New flat package `internal/release` (peer to
  `version`/`config`/`server`), **stdlib only**. Models the P3 metadata exactly:
  `Version`, `CommitSHA`, `PreviousCommitSHA`, `CreatedAt`, `Status`.
- **State machine**: `Status` with the five PLAN states and a linear transition
  table `Draft → Created → Staging → Production → Archived`. `status` is guarded
  (unexported + `Status()` getter); `TransitionTo` is the only mutator and
  enforces the table. `New` validates and starts at `Draft`, with `createdAt`
  injected (no hidden clock).
- **Tests**: table-driven, stdlib `testing` (keeps the zero-deps invariant — no
  testify). Cover `New` validation, valid/invalid transitions, full lifecycle.
- **Deferred** (phase boundary): no repository port (P5, with the pgx impl — no
  port without an adapter), no DB/migrations (P5), no `release` CLI (P6), no HTTP
  wiring. `GET /version` untouched. Rollback/abandon transitions are P6 semantics
  and intentionally not modelled.
- Package doc in `docs/module/release.md` (project pref: no in-code godoc).

## 2026-06-19 Update — Phase 4 scaffold (changelog generation)

- New flat package `internal/changelog` (peer to `release`/`version`), **stdlib
  only**. Pure: turns conventional-commit subjects into grouped markdown notes,
  no git, no AI, no persistence.
- **API**: `Parse(subject) (Entry, bool)` parses one `type(scope)?!?: desc`
  subject (type lowercased, scope + breaking `!` stripped, `ok=false` for
  non-conventional lines); `Generate([]string) string` groups by type and renders
  `feat → Features`, `fix → Fixes`, `refactor → Improvements` in that order.
  Unmapped types and noise are skipped; empty input → `_No notable changes._`.
- **Git deferred**: `git log previous_sha..current_sha` is the intended source,
  but `Generate` takes a plain `[]string` so it stays pure/unit-testable. The git
  exec lands with the P6 CLI (`release create`) that calls it — same precedent as
  P3 deferring its repository adapter (no untested I/O glue this phase).
- **No `release.go` change**: persistence (`releases.changelog`) is P5; P4 only
  produces the string. No public contract or runtime behavior change.
- **Tests**: table-driven, stdlib `testing` (zero-deps invariant — no testify).
  Cover parse (scope/`!`/case/noise) and generate (ordering, drop, empty).
- Package doc in `docs/module/changelog.md` (project pref: no in-code godoc).

## 2026-06-19 Update — Phase 5 scaffold (deployment tracking)

- New domain package `internal/deployment` (peer to `release`): `Deployment`
  entity (`ID`, `ReleaseID`, `Environment`, `Status`, `DeployedAt`), `New`
  (opens at `pending`) + `Rehydrate` for DB reads, and a `Repository` port.
- `internal/release` gained an additive `ID int64` + `Changelog` field, a
  `Rehydrate` constructor (sets `status` directly — DB reads bypass the state
  machine; `New` stays Draft-only, guard holds for live objects), `ErrNotFound`,
  and a `Repository` port. No behavior change to `New`/`TransitionTo`.
- **First external dependency**: `github.com/jackc/pgx/v5`. New `internal/postgres`
  package is the single adapter implementing both ports (`Store` aggregating
  `releaseRepo`/`deploymentRepo` — split because both ports declare `Create`).
  Per-call timeout, `RETURNING id`, `pgx.ErrNoRows → domain ErrNotFound`.
- **Migrations**: plain `migrations/000001_create_releases.{up,down}.sql` and
  `000002_create_deployments.{up,down}.sql` (golang-migrate naming, no tool/lib
  added). Schema matches PLAN exactly + CHECK guards on status/environment, FK
  `deployments.release_id → releases.id`, index `(environment, deployed_at)`.
- **Storage placement** (per user): Tsugi reuses the box's existing shared
  Postgres (same instance as LazyScan) in a `tsugi` database — no dedicated
  container. One instance holds both environments' history (`environment`
  column), since deployment history must be queryable across staging+prod.
- **Deferred to P6**: DB wiring in `cmd/tsugi` (`TSUGI_DATABASE_URL` + pool),
  migration runner, deployment outcome transitions (`pending → succeeded/failed`),
  `WithTx`/DBTX (no atomic multi-write yet), and DB integration test (needs a
  live Postgres). `GET /version`/`/healthz` untouched; service still starts
  without a DB.
- **Doc drift flagged**: architecture.md earlier said the repository "port lands
  in P5 with the pgx impl" — done as stated. The `release` doc note that there
  was "no repository port yet" is now stale and updated.

## 2026-06-19 Update — Phase 6 scaffold (promotion & rollback)

- The `release` CLI lands: `cmd/tsugi` gains subcommand dispatch — `serve`
  (unchanged), `migrate up|down`, and `release create|list|show|promote|rollback`.
  `serve` still starts without a DB; only the CLI paths open a pool.
- **New flat packages** (peers, not a DDD subtree): `internal/cli` (use-cases),
  `internal/git` (`os/exec` wrapper — P4's deferred git source), `internal/deploy`
  (shells `deploy.sh`). `internal/postgres` gains `Connect`, `WithTx` (the
  deferred tx seam), `UpdateStatus`, and an embedded migration runner.
- **Semantics**: `create vX` snapshots the validated staging commit + changelog,
  records the release at `Staging`. `promote vX` runs the real prod deploy at the
  release commit, then atomically advances `Staging → Production`, archives the
  prior production release, and marks the deployment succeeded. A failed deploy
  marks the deployment `failed` and leaves the release at staging. `rollback vX`
  re-deploys an archived release via the new `Archived → Production` edge.
- **Deploy-by-SHA**: `deploy.sh` gained `--ref <sha>` (charset-guarded) — a
  detached checkout instead of `pull --ff-only`, so rollback lands an older
  commit. The prod←`main` / staging←`dev` invariant holds on the normal path.
- **Migrations**: applied with `tsugi migrate up` (embedded `*.sql`, tracked in
  `schema_migrations`); `down` rolls back the last step. Minimal — no dirty-state
  recovery. The `no golang-migrate lib` decision from P5 holds.
- **Tests**: rollback edge (`release`), outcome mutators (`deployment`), and full
  create/promote/rollback orchestration in `internal/cli` via in-memory fakes
  (no live DB/git/docker). The pgx adapter + runner still need a live Postgres —
  validated by compile + `go vet`.

## 2026-06-20 — Phase 7 (target topology correction) — **repo sweep done, VPS pending**

Cleanup/refactor phase. Not in the original PLAN — added because the P1 deploy
scaffold and parts of the wiki were authored against the **wrong target**.

- **Root cause (negligence):** P1 modeled the deploy layer on the legacy
  `../LazyScan-Stack` (detached services: `aegis` edge, `minio`, separate
  `s3.<domain>` hosts, `docker-compose.prod.yml`). The live target is the trimmed
  monorepo `../LazyScan`: a single nginx `web` service (SPA + `/api` proxy) on
  `127.0.0.1:8080:80`, internal `api`/`image-svc`/`mail-svc`/`postgres`/`redis`,
  storage on **R2** (presigned PUT browser→R2, not through Tsugi's tunnel), one
  `docker-compose.yml`. The Go service/CLI/postgres code is target-agnostic and is
  **not** affected — this is config + docs only.
- **Corrected topology (from live VPS, 2026-06-20):** one tunnel hostname per env,
  one nginx `web` edge per stack, no s3 host, no minio.
  - prod: `lazyscan.my.id` → `127.0.0.1:8081` — **already live** (`lazyscan-web-1`,
    compose project `lazyscan`). Untouched by P7.
  - staging: `staging.lazyscan.my.id` → `127.0.0.1:8082` — **new** (compose project
    `lazyscan-staging`). 8080 is taken by `dozzle`, so staging takes 8082.
  - App lives in its own zone `lazyscan.my.id`; infra hosts are under
    `*.sanctuary.my.id`. Tunnel `vps` (`e7f99e75-…`), config
    `/etc/cloudflared/config.yml`.
- **Compose project alignment:** the live prod stack runs as bare project
  `lazyscan` (started with `docker compose up` in `../LazyScan`). `deploy.sh`
  currently names projects `<target>-<env>` → it would spawn a *duplicate*
  `lazyscan-prod`. Fix: prod uses the bare target name (`lazyscan`), staging uses
  `<target>-staging` (`lazyscan-staging`).
- **Scope (config + docs):**
  - `deploy/bin/deploy.sh` — project name: prod = `<target>`, staging =
    `<target>-staging` (was `<target>-<env>`).
  - `deploy/targets/lazyscan/target.env` — `BASE_COMPOSE=docker-compose.yml`,
    checkout paths point at `../LazyScan` clones (real prod-checkout path TBD —
    confirm on VPS).
  - `docker-compose.{prod,staging}.override.yml` — service `web`, port `:80`
    (`8081:80` / `8082:80`), drop the `minio` block.
  - `cloudflared/config.yml.example` — prod `lazyscan.my.id`→8081, staging
    `staging.lazyscan.my.id`→8082; drop both `s3` rules and the `aegis` framing.
  - `.env.{prod,staging}.example` — `PUBLIC_APP_URL` = `https://lazyscan.my.id` /
    `https://staging.lazyscan.my.id`; drop `PUBLIC_S3_URL`.
  - `docs/wiki/{architecture,running,known-constraints,infra-plan}.md` — topology,
    port map (8081/8082), hosts, packages note (`aegis`→`web`), drop minio/s3.
- **VPS action (not in-repo):** append one ingress rule
  (`staging.lazyscan.my.id` → `127.0.0.1:8082`) above the `404` catch-all in
  `/etc/cloudflared/config.yml`, then `cloudflared tunnel route dns vps
  staging.lazyscan.my.id`. Prod rule untouched.
- **Success criteria:** deploy scaffold + wiki match the live `../LazyScan`
  topology exactly; `deploy.sh --env prod` targets the existing `lazyscan` project
  (no duplicate); staging reachable at `staging.lazyscan.my.id`.

Original plan below kept as-is for history.

---

# Release Promotion Platform

## Goal

Provide a controlled deployment workflow with release visibility, changelog
generation, promotion, and rollback support.

The system should answer:

- What version is running?
- What commit is deployed?
- What changed?
- When was it deployed?
- Can it be rolled back?

## Current State

Branches: `main`, `dev`. Infrastructure: single VPS.

Problem: production currently runs code from `dev`, making every push a
potential production deployment.

## Target Architecture

Branches `main`, `dev`. Environments staging + production. No staging branch —
environment separation is sufficient.

- staging → deploys from `dev`
- production → deploys from `main`

## Infrastructure

Single VPS. Containers: `api-prod` (branch `main`, production traffic),
`api-staging` (branch `dev`, validation traffic). Routing: `api.domain.com` →
production, `staging-api.domain.com` → staging. Safe validation without a second
VPS.

## Workflow

```txt
Local Development → Push to dev → Deploy to Staging → Validate
→ Create Release → Generate Changelog → Merge dev → main → Deploy Production
```

Production deployments should always originate from `main`.

## Phase 1 — Environment Separation

Objective: prevent untested code from reaching production.

Deliverables: staging container, production container, separate
domains/subdomains, separate deployment targets.

Success criteria: production is no longer running `dev`.

## Phase 2 — Version Visibility

Objective: know exactly what is running.

`GET /version` →

```json
{ "version": "v1.2.0", "commit": "abc123", "deployed_at": "2026-06-19T20:00:00Z" }
```

Success criteria: every deployment is identifiable.

## Phase 3 — Release Management

Objective: track releases instead of branches.

Release metadata: version, commit SHA, previous commit SHA, created at, status.

Release states: Draft → Created → Staging → Production → Archived.

Success criteria: a deployment references a release, not a branch.

## Phase 4 — Changelog Generation

Objective: automatically document changes.

Input: `git log previous_sha..current_sha`. Requirements: conventional commits,
no AI dependency. Groups `feat`/`fix`/`refactor` into Features/Fixes/
Improvements.

Success criteria: every release contains release notes.

## Phase 5 — Deployment Tracking

Objective: maintain deployment history.

```txt
releases     (id, version, commit_sha, previous_commit_sha, changelog, status, created_at)
deployments  (id, release_id, environment, status, deployed_at)
```

Success criteria: deployment history is queryable.

## Phase 6 — Promotion & Rollback

Objective: promote validated releases safely.

Commands: `release create v1.2.0`, `release list`, `release show v1.2.0`,
`release promote v1.2.0`, `release rollback v1.1.5`.

Promotion: Release → Deploy Staging → Validate → Promote → Deploy Production.
Rollback: Production → Previous Stable Release.

Success criteria: production can be reverted without manual git operations.

## Future Enhancements

Build once / deploy many, artifact storage, health checks, automatic rollback,
multi-VPS deployment, Discord notifications, deployment audit trail.

## Non-Goals

Kubernetes replacement, GitHub Actions replacement, ArgoCD clone, multi-tenant
platform, enterprise CI/CD. The objective is operational safety and deployment
visibility for personal projects, not platform engineering for its own sake.
