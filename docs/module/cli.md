# internal/cli

P6 application layer: the `release` CLI use-cases — `create`, `list`, `show`,
`promote`, `rollback`. Orchestrates the domain (`release`/`deployment`),
`changelog`, git history, and the deployer. Delivery + wiring is `cmd/tsugi`.

## Symbols

- `App` (struct) — injected deps: `Releases`/`Deployments` ports, `Tx`
  (`TxRunner`), `Git` (`GitReader`), `Deployer`, `Target`, `StagingCheckout`,
  optional `Now` (clock), `Out`.
- `App.Run(ctx, args)` — dispatches the subcommand.
- `GitReader`, `Deployer`, `TxRunner` (interfaces) — the side-effecting seams,
  faked in tests so the orchestration runs without a DB, git, or docker.

## Commands

- `create vX` — reads the staging checkout HEAD, generates the changelog from
  `prev..head`, records the release at **Staging** (the validated commit is
  already on staging per `PLAN.md`'s workflow).
- `list` / `show vX` — read-only views.
- `promote vX` — staging → production. Records a pending production deployment,
  runs the real deploy at the release commit, then advances release status +
  archives the previous production release + marks the deployment succeeded —
  atomically (`TxRunner.WithTx`). A failed deploy marks the deployment `failed`
  and leaves the release at staging (state advances only on success).
- `rollback vX` — re-deploys an **archived** release to production via
  `deploy.sh --ref <sha>` (uses the `Archived → Production` lifecycle edge).

## Notes

- The composition root (`cmd/tsugi`) supplies the real `git.Default`,
  `deploy.Script`, and `postgres.Store`. `App` itself touches no I/O directly.
- Single-operator assumptions: no row locking; the in-memory release pointer is
  reused across the deploy (no re-fetch inside the tx).
