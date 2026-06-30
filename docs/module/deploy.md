# internal/deploy

P6 adapter that shells out to `deploy/bin/deploy.sh` — Tsugi orchestrates the
target's compose, it does not reimplement it in Go.

## Symbols

- `Run(ctx, binDir, target, env, ref, sink)` — runs `deploy.sh --target --env
  [--ref]`; a non-empty `ref` deploys that commit (rollback), else the env
  branch HEAD. Streams stdout/stderr to `sink` (`deployflow.LogSink`) line by line.
- `Script` (struct) — adapts `Run` to `deployflow.Deployer`, bound to `deploy/bin`.
- `StagingCheckout(deployDir, target)` — reads `CHECKOUT_STAGING` from the
  target's `target.env` (the dev checkout the CLI reads git history from).

## Notes

- `--ref` is the SHA-aware path `deploy.sh` gained for P6: with a ref it does a
  detached checkout instead of the `pull --ff-only` branch path, so rollback can
  land an older commit. Still main lineage for prod — the prod←`main` /
  staging←`dev` invariant holds on the normal (no-ref) path.
- The actual `docker compose up` runs on the VPS; locally it just invokes the
  script (which guards inputs and fails fast when a checkout/env-file is absent).
