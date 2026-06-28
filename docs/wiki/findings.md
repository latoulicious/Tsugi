# Findings

Code-review findings log. Paired with [`resolutions.md`](resolutions.md) — every
finding that gets fixed must have a matching resolution entry; the two are
interconnected and must not be orphaned.

Format per finding:

```md
## F-NNN <short title>
- date:
- source: <review tool / PR / manual>
- severity: low | medium | high
- location: path:line
- problem:
- status: open | resolved (→ R-NNN)
```

## F-001 Unvalidated `--target` flows into path + `source`
- date: 2026-06-19
- source: CodeRabbit CLI (`coderabbit review --agent`), Phase 1 scaffold
- severity: medium (reported major)
- location: deploy/bin/deploy.sh:38,43
- problem: `TARGET` from `--target` is concatenated into
  `TARGET_DIR="$DEPLOY_DIR/targets/$TARGET"` and then `source`d. A traversal
  value (e.g. `../../x`) could read/source a file outside `targets/`.
- status: resolved (→ R-001)

## F-002 HEALTHCHECK `wget` assumed missing in alpine:3.23
- date: 2026-06-19
- source: CodeRabbit CLI (`coderabbit review --agent`), Phase 2 scaffold
- severity: low (reported critical)
- location: Dockerfile:31
- problem: review claims `wget` is absent in `alpine:3.23`, so the healthcheck
  would fail.
- status: rejected (false positive) — alpine's busybox ships the `wget` applet;
  Aegis + Herald use the identical `wget -qO-` healthcheck on `alpine:3.23` with
  no `apk add`. Keeping it is stack parity. Re-run reported 0 findings.

## F-003 HTTP server missing `WriteTimeout`
- date: 2026-06-19
- source: CodeRabbit CLI (`coderabbit review --agent`), Phase 2 scaffold
- severity: medium (reported major)
- location: internal/server/server.go:26
- problem: `http.Server` set `ReadHeaderTimeout`/`IdleTimeout` but no
  `WriteTimeout`; a slow-reading client could hold a connection open.
- status: resolved (→ R-002)

## F-004 `DeployedAt` field name misleading
- date: 2026-06-19
- source: CodeRabbit CLI (`coderabbit review --agent`), Phase 2 scaffold
- severity: low (reported minor)
- location: internal/version/version.go:22
- problem: field carries build time, not literal deployment time; review
  suggested renaming to `build_date`.
- status: rejected (contract) — `PLAN.md:144,258` mandate the JSON key
  `deployed_at`; renaming breaks the Phase 2 spec. Build≈deploy semantics are
  documented in `docs/module/version.md`.

## F-005 `/version` may emit a half-written 200 on encode failure
- date: 2026-06-19
- source: CodeRabbit CLI (`coderabbit review --agent`), Phase 2 scaffold
- severity: low (reported minor)
- location: internal/server/server.go:34
- problem: streaming `json.NewEncoder(w).Encode` writes the 200 header before a
  possible encode error, leaving an incomplete body under a success status.
- status: resolved (→ R-003)

## F-006 `Rehydrate` returns `ErrInvalidReleaseID` for an invalid `id`
- date: 2026-06-19
- source: CodeRabbit CLI (`coderabbit review --agent`), Phase 5 scaffold
- severity: low (reported minor)
- location: internal/deployment/deployment.go:68
- problem: `Rehydrate` collapses `id <= 0 || releaseID <= 0` into one check that
  always returns `ErrInvalidReleaseID`, so a bad `id` is misreported as a bad
  `releaseID`. Both come from the DB row, so it is a defensive guard, not a
  user-facing path.
- status: resolved (→ R-004)

## F-007 Wiki port docs stale after align-to-live (8081/8082 → 8080/8090)
- date: 2026-06-28
- source: manual (deploy topology reconciliation session)
- severity: low
- location: docs/wiki/{architecture,running,known-constraints,infra-plan}.md
- problem: canonical port scheme was realigned to the live tunnel — prod web
  8080, staging web 8090 (prod/staging overrides + cloudflared example updated in
  c8b84f2). The current-state wiki tables still cite the old 8081/8082. Dated
  session logs (19/20-06) left as history, not edited.
- status: open

## F-008 Port 8090 double-booked: staging `web` vs `tsugi serve`
- date: 2026-06-28
- source: manual
- severity: medium
- location: deploy/targets/lazyscan/docker-compose.staging.override.yml; serve default port
- problem: staging LazyScan `web` now publishes `127.0.0.1:8090` to match the
  cloudflared staging ingress. The journal reserves 8090 for `tsugi serve`. No
  live clash today (serve not running), but running serve while staging is up
  collides. Pick a distinct serve port (or move staging) before running serve on
  the box.
- status: open
