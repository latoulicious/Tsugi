# internal/deployment

P5 domain entity: one release deployed to one environment, plus the repository
port for deployment history. Pure domain — stdlib only; the pgx adapter lives in
[postgres.md](postgres.md).

## Symbols

- `Environment` (string) — `EnvStaging` (`staging`), `EnvProduction`
  (`production`). `Valid()` reports a known value.
- `Status` (string) — `StatusPending`, `StatusSucceeded`, `StatusFailed`.
  `Valid()` reports a known value.
- `Deployment` (struct) — `ID`, `ReleaseID` (int64), `Environment`, `Status`,
  `DeployedAt`. `ID` is 0 until persisted (store assigns via `RETURNING`).
- `New(releaseID, env, deployedAt) (*Deployment, error)` — validates and starts
  the record at `StatusPending`.
- `Rehydrate(id, releaseID, env, status, deployedAt) (*Deployment, error)` —
  reconstructs a persisted row for DB reads (validates all fields + status).
- `Repository` (interface) — `Create`, `List`, `ListByEnvironment`. Consumer-
  defined port; implemented by `internal/postgres`.
- Sentinel errors: `ErrInvalidID`, `ErrInvalidReleaseID`,
  `ErrInvalidEnvironment`, `ErrInvalidStatus`, `ErrNotFound`.

## Notes

- A deployment references a release by `ReleaseID`, not a branch — the P3/P5
  point. The FK is enforced at the DB (`deployments.release_id`).
- Outcome transitions (`pending → succeeded`/`failed`) are **not** modelled
  here; the deploy executor sets the final status in P6. `New` only opens the
  record at `pending`.
- `Environment` is restricted to the two PLAN environments and CHECK-guarded in
  the migration, mirroring the prod←`main` / staging←`dev` invariant.
