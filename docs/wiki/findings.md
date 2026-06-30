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

## F-009 gRPC GracefulStop has no shutdown deadline
- date: 2026-06-29
- source: coderabbit (P5.2 review)
- severity: medium
- location: cmd/tsugi/main.go:115-118
- problem: `run()` calls `grpcSrv.GracefulStop()` with no bound; a slow or hung
  RPC blocks shutdown indefinitely, and the existing `shutdownTimeout` only
  covers `httpSrv.Shutdown`. Should bound GracefulStop by `shutdownCtx` and fall
  back to `grpcSrv.Stop()` so HTTP shutdown + process exit still complete.
- status: resolved (→ R-005)

## F-010 HTTP `TSUGI_ADDR` not loopback-guarded like the agent
- date: 2026-06-29
- source: coderabbit (P5.2 review)
- severity: low
- location: internal/config/config.go:23-31
- problem: only `AgentAddr` is loopback-validated; `TSUGI_ADDR` may bind a public
  interface. Assessment: **by design** — the HTTP `/version`+`/healthz` plane is
  meant to be fronted by the Cloudflare tunnel (known-constraints); only the
  write-plane agent must stay off-box. Forcing loopback on `TSUGI_ADDR` would
  break the documented topology and remove a legitimate override. Recommend
  won't-fix; documenting the rationale here.
- status: wontfix (by design — HTTP plane is tunnel-fronted)

## F-011 `requireLoopback` trusts the literal host `localhost`
- date: 2026-06-29
- source: coderabbit (P5.2 review)
- severity: medium
- location: internal/config/config.go:41-52
- problem: special-casing the string `localhost` relies on name resolution; a
  misconfigured/hostile `/etc/hosts` could map it off-loopback and defeat the
  write-plane guarantee. Tightening to literal loopback IPs only
  (`net.ParseIP(host).IsLoopback()`) removes the DNS vector and is less code.
- status: resolved (→ R-006)

## F-012 `requireLoopback` does not validate the port range
- date: 2026-06-29
- source: coderabbit (P5.2 review)
- severity: low
- location: internal/config/config.go:43-46
- problem: `net.SplitHostPort` accepts an out-of-range/non-numeric port; a bad
  value is caught at `net.Listen` (startup) rather than at config load.
  Assessment: `net.Listen` already fails fast with a clear error, and the
  existing `TSUGI_ADDR` validation is equally lenient — low value. Recommend
  defer (YAGNI) unless symmetric strict parsing is wanted.
- status: deferred (net.Listen fail-fasts; revisit with symmetric Addr parsing)

## F-013 read RPCs query releases + deployments non-atomically
- date: 2026-06-29
- source: coderabbit (P5.2 review)
- severity: low
- location: internal/agent/server.go:47-90
- problem: `ListReleases`/`ListDeployments` issue two independent `List()` reads,
  not one snapshot; a write landing between them can skew the denormalized view
  (e.g. a deployment whose release post-dates the releases read → empty commit).
  The `deployments.release_id` FK rules out a true orphan, so the only cause is
  read-skew, and it self-heals on the next poll of this read-only overview.
  Options: a consistent read-tx snapshot (REPEATABLE READ) vs accept + document
  for a polling read plane.
- status: accepted (read-skew self-heals; FK prevents true orphans)

## F-014 deployflow success line streamed inside the WithTx closure
- date: 2026-06-30
- source: coderabbit (P5.4 review)
- severity: major
- location: internal/deployflow/deployflow.go:71-89
- problem: `ToProduction` emitted the "deployed … to production" line via
  `sink.Line` *inside* the `WithTx` callback, before the transaction committed.
  Harmless when the sink was the CLI's terminal, but P5.4 streams it to a browser:
  a commit failure would show "deployed" and then an error line. Move the success
  report after `WithTx` returns nil.
- status: resolved (→ R-007)

## F-015 ToProduction does not pre-validate the release transition
- date: 2026-06-30
- source: coderabbit (P5.4 review)
- severity: major
- location: internal/deployflow/deployflow.go:54-62
- problem: suggestion to pre-check `r.TransitionTo(StatusProduction)` on a copy
  before `Deployments.Create`/`Deployer.Run` so an invalid state fails fast.
  Assessment: both callers already guard status before `ToProduction` — the agent
  rejects non-`staging`/`archived` (`FailedPrecondition`), the CLI rejects in
  `Promote`/`Rollback` — and the lifecycle map allows exactly `staging→production`
  and `archived→production`. So `ToProduction` is only ever reached with a
  transition-valid release; a pre-check duplicates the existing guard for no gain.
- status: wontfix (callers already guard status; lifecycle map covers the edge)

## F-016 known-constraints mixes `dev` and `development`
- date: 2026-06-30
- source: coderabbit (P5.4 review)
- severity: minor
- location: docs/wiki/known-constraints.md
- problem: the page uses both `dev` (Single VPS section) and `development`
  (branch-invariant section) for the staging branch. Assessment: pre-existing
  wording predating P5.4; tidying it is unrelated cleanup outside this slice's
  scope (working rules: no unrelated cleanup). The branch invariant section is
  unambiguous (`staging deploys development`).
- status: wontfix (pre-existing; unrelated-cleanup, out of slice scope)
