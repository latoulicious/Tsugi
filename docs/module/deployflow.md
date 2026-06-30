# internal/deployflow

The production-deploy use case, shared by the `release` CLI and the write-plane
`agent` so the promotion state machine lives in exactly one place. Extracted from
`cli` in P5.4 when the agent needed the same orchestration to stream over gRPC.

## Symbols

- `Service` (struct) — injected deps: `Deployments` port, `Tx` (`TxRunner`),
  `Deployer`, `Target`, optional `Now` (clock).
- `Service.ToProduction(ctx, r, sink)` — records a pending production deployment,
  runs the real deploy at `r.CommitSHA` (streaming to `sink`), then on success
  archives the previous production release + advances `r` to production + marks the
  deployment succeeded, atomically (`TxRunner.WithTx`). On deploy failure it marks
  the deployment `failed` and returns the error — the release stays put.
- `Deployer`, `TxRunner`, `LogSink` (interfaces) — the side-effecting seams (real
  impls: `internal/deploy`, `postgres.Store`; sinks: the CLI's `WriterSink` and the
  agent's gRPC stream adapter).
- `WriterSink` (struct) — adapts an `io.Writer` to `LogSink`, one line per write
  (the CLI's terminal sink).

## Notes

- Single-active-production: `archivePrevious` demotes any other release in
  production before advancing this one (only one production release at a time).
- The in-memory release pointer is reused across the deploy (no re-fetch inside the
  tx) — single-operator assumption, no row locking.
