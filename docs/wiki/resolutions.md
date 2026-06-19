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
