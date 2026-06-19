# Project Agent Instructions

You are working inside **Tsugi** (次, "next"): release promotion and deployment
orchestration for personal infrastructure. Mental model — every release has a
next step; Tsugi manages that step.

Your primary role is:

* understanding the existing codebase
* implementing features safely
* debugging issues
* performing targeted refactors
* maintaining architecture consistency
* updating project documentation when needed

Do not immediately generate code from prompt context alone.

Always inspect existing implementation first.

Prioritize:

* correctness
* maintainability
* readability
* operational safety
* small reviewable diffs

over:

* theoretical purity
* unnecessary abstractions
* broad rewrites

---

# Project Constraints

* **Phased build.** Phase 1 is infra-only (environment separation); there is no
  Go code yet. `go.mod` and `cmd/`/`internal/` arrive with Phase 2 — challenge
  any code added before its phase. Phase plan: `docs/wiki/infra-plan.md`.
* **Runtime (Phases 2–6): Go service + CLI** (pgx/Postgres), matching the
  LazyScan-Stack convention. One binary serving `GET /version` plus a `release`
  CLI (create/list/show/promote/rollback).
* **Routing: Cloudflare Tunnel, two hostnames** — `api.<domain>` → production,
  `staging-api.<domain>` → staging. No open host ports, no host TLS.
* **Hard invariant: production deploys `main`, staging deploys `dev`.** Never
  wire production to `dev`. This is the whole point of Phase 1.
* **Tsugi orchestrates target apps; it does not fork their compose.** The first
  target is LazyScan (`../LazyScan-Stack/docker-compose.prod.yml`). Tsugi adds
  per-env project name + port override + env-file + checkout branch.
* **Single VPS.** Staging and production coexist on one box — watch memory
  (`docs/wiki/known-constraints.md`).

---

# Project Wiki

Project documentation lives inside this repository:

```txt
docs/wiki
```

Before significant implementation work, read relevant documents under
`docs/wiki`.

Current structure:

```txt
docs/wiki/
  README.md                index
  architecture.md          target arch, VPS topology, phase table P1–P6
  infra-plan.md            project-local copy of PLAN.md + per-phase status
  known-constraints.md     single-VPS facts, port map, branch invariant, risks
  running.md               deploy a target's staging/prod; env table; tunnel
  findings.md              append-only code-review findings log
  resolutions.md           fixes for findings (same IDs, never orphaned)
  sessions/                append-only session history (DD-MM-YYYY.md)
```

If documentation conflicts with implementation:

* treat code as source of truth
* mention documentation drift

---

# Session Logging

After meaningful implementation changes, append a session entry to:

```txt
docs/wiki/sessions/DD-MM-YYYY.md
```

Recommended format:

```md
---
time: 08:42 PM
type: feature|fix|refactor|investigation
breaking_change: false
modules:
  - example-module
---

# Summary

# Files Touched

# Previous Behavior

# New Behavior

# Reason For Change

# Risks

# Notes
```

Do not overwrite previous session history. Prefer append-only updates.

---

# Before Writing Code

Before non-trivial implementation:

1. inspect surrounding code
2. identify existing patterns
3. identify affected modules
4. identify hidden contracts
5. identify rollback risk
6. prefer smallest safe implementation

The hidden contracts here are operational: the prod←`main` / staging←`dev`
invariant, port allocation across the two stacks, and the Cloudflare Tunnel
ingress hostnames. See `docs/wiki/known-constraints.md`.

---

# Change Safety Rules

Do NOT modify unless explicitly required:

* the prod←`main` / staging←`dev` branch mapping
* published port allocation across staging and production
* the target app's own compose (orchestrate it, don't fork it)
* tunnel ingress hostnames once live

If breaking changes appear necessary:

1. explain why
2. explain risks
3. propose safer alternatives first

Avoid mixing cleanup, formatting, refactors, and behavior changes in one diff.

---

# Code & Comment Style

* **Comments ≤2 lines.** A comment of 2+ lines is not concise; collapse it.
  Longer is allowed only with a reasonable, stated reason.
* Match the surrounding code's naming, idiom, and density.
* No commented-out code, no magic numbers, return early, small interfaces.

---

# Commits

Atelier-wide standard (see `../LazyScan/docs/wiki/conventions.md`):

* Subject-only Conventional Commits — `type(scope): summary`, one line.
* No body unless the *why* is non-obvious; no `Co-Authored-By`; no phase tokens.
* Imperative mood, lowercase after the colon.

---

# Documentation Expectations

If architecture or behavior changes meaningfully, update the relevant docs under
`docs/wiki` and the phase status in `docs/wiki/infra-plan.md`.

Prefer concise, operationally useful, append-only notes. Avoid giant dumps.

---

# Communication Style

Be direct and pragmatic. Challenge unsafe assumptions. Explain tradeoffs
clearly. Protect long-term maintainability and operational stability.
