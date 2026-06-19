# Tsugi Infra Plan

Project-local copy of the original `PLAN.md` (Release Promotion Platform),
implemented as **Tsugi**. Keep future per-phase status here; the original plan is
kept verbatim below for history.

Type: Go release-promotion + deployment-orchestration service + CLI.

Status: **P2 scaffold done** — `GET /version` service (2026-06-19). P1
environment-separation scaffold (2026-06-19).

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
