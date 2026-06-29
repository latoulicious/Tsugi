# Resolutions

How findings were resolved. Paired with [`findings.md`](findings.md) — each entry
resolves a specific finding and must reference it; neither file is orphaned.

Format per resolution:

```md
## R-NNN <short title>  (resolves F-NNN)
- date:
- change: <what was done>
- files:
- verification: <how it was confirmed>
- constraints honored: <Do-Not rules respected>
```

## R-001 Guard `--target` against traversal  (resolves F-001)
- date: 2026-06-19
- change: Guard `--target` with `^[A-Za-z0-9_-]+$` right after arg parse and
  exit 2 on mismatch, before `TARGET` reaches any path or `source`. Blocks
  traversal/odd chars; valid targets unaffected. `--env` was already whitelisted.
- files: deploy/bin/deploy.sh (guard after the missing-arg check)
- verification: `bash -n` clean; `--target ../../x --env prod` → "invalid
  --target" exit 2; valid target unchanged; CodeRabbit re-run clean (0 findings).
- constraints honored: smallest safe change; no public-contract / behavior change
  for valid input; no unrelated cleanup; comment ≤2 lines.

## R-002 Add `WriteTimeout` to the HTTP server  (resolves F-003)
- date: 2026-06-19
- change: Added a `writeTimeout` const (30s) and set `WriteTimeout` on the
  `http.Server` alongside the existing `ReadHeaderTimeout`/`IdleTimeout`.
- files: internal/server/server.go
- verification: `go vet` clean; `make build` ok; `/version` + `/healthz` still
  return expected JSON; CodeRabbit re-run → 0 findings.
- constraints honored: smallest safe change; no contract/behavior change for
  normal requests; no unrelated cleanup.

## R-003 Pre-marshal `/version` before writing  (resolves F-005)
- date: 2026-06-19
- change: `handleVersion` now `json.Marshal`s `version.Get()` to a buffer; on
  error it returns `500`, otherwise sets the content-type and writes the body
  (default 200). No partial body under a success status.
- files: internal/server/server.go
- verification: `go vet` clean; `make build` ok; `/version` returns the full
  JSON payload; CodeRabbit re-run → 0 findings.
- constraints honored: smallest safe change; `deployed_at` contract unchanged;
  comment ≤2 lines; no unrelated cleanup.

## R-004 Distinct `ErrInvalidID` in `Rehydrate`  (resolves F-006)
- date: 2026-06-19
- change: Added `ErrInvalidID` sentinel and split the `Rehydrate` guard so
  `id <= 0` returns it and `releaseID <= 0` returns `ErrInvalidReleaseID`,
  instead of one combined check reporting the release-id error for both.
- files: internal/deployment/deployment.go, internal/deployment/deployment_test.go
- verification: `go vet` clean; `go test ./internal/deployment/` green (added a
  zero-`releaseID` case alongside the zero-`id` case); `gofmt` clean.
- constraints honored: smallest safe change; `New` path untouched; no public
  behavior change beyond the more precise error; no unrelated cleanup.

## R-005 Bound gRPC shutdown by the deadline  (resolves F-009)
- date: 2026-06-29
- change: Added `stopGRPC(ctx, srv)` — runs `GracefulStop` in a goroutine and
  falls back to `Stop()` when `shutdownCtx` (the existing `shutdownTimeout`)
  expires, so a slow RPC can't stall HTTP shutdown or process exit.
- files: cmd/tsugi/main.go
- verification: `go build`/`vet` clean; `gofmt` clean; SIGTERM still logs
  "shutdown complete" (graceful path, reads drain instantly).
- constraints honored: smallest safe change; reuses existing `shutdownTimeout`;
  no contract change; serve behavior otherwise untouched.

## R-006 Loopback IPs only, drop `localhost`  (resolves F-011)
- date: 2026-06-29
- change: `requireLoopback` now accepts only a literal loopback IP
  (`net.ParseIP(host).IsLoopback()`) — dropped the `localhost` string branch so
  the write-plane guard no longer depends on name resolution. Error message now
  reads "127.0.0.1 or ::1".
- files: internal/config/config.go
- verification: `go build`/`vet`/`test ./internal/agent` green; `gofmt` clean;
  `TSUGI_AGENT_ADDR=localhost:8091` now rejected at config load, `127.0.0.1`/`::1`
  pass; `0.0.0.0` still rejected.
- constraints honored: tightens a security boundary with less code; no public
  contract change; default `127.0.0.1:8091` unaffected.
