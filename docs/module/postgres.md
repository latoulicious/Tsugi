# internal/postgres

P5 infrastructure adapter: pgx-backed implementation of the `release` and
`deployment` repository ports. First package with an external dependency
(`github.com/jackc/pgx/v5`); everything else stays stdlib. P6 added the tx seam,
a pool constructor, and the migration runner.

## Symbols

- `Store` (struct) — aggregates the two repositories: `Releases`
  (`release.Repository`) and `Deployments` (`deployment.Repository`).
- `New(pool *pgxpool.Pool) *Store` — wires both sub-repos over one pool.
- `Connect(ctx, url) (*pgxpool.Pool, error)` — opens a pool (caller owns Close).
- `Store.WithTx(ctx, fn)` — runs `fn(release.Repository, deployment.Repository)`
  against tx-scoped repos, committing on success. The P6 promotion/rollback
  atomic write (release status + deployment outcome together).
- `DBTX` (interface) — the `Query`/`QueryRow`/`Exec` surface shared by the pool
  and a `pgx.Tx`, so a repo runs the same code in or out of a transaction.
- `MigrateUp` / `MigrateDown(ctx, pool)` — apply pending `*.up.sql` (each in its
  own tx, tracked in `schema_migrations`) / roll back the last one. Migration
  files are embedded via `migrations.FS` (no path dependency at runtime).
- `releaseRepo` / `deploymentRepo` (unexported) — concrete pgx impls over `DBTX`.
  Split into two types because both ports declare `Create`. Each adds
  `UpdateStatus` for the P6 status writes.

## Behavior

- Each call sets a `queryTimeout` (5s) via `context.WithTimeout`.
- `Create` inserts with `RETURNING id` and writes the new id back onto the
  entity (`rel.ID` / `dep.ID`).
- Reads go through the domain `Rehydrate` constructors, so a row that violates a
  domain invariant fails the scan rather than producing a bad entity.
- `pgx.ErrNoRows` maps to the domain `ErrNotFound` at the boundary; other errors
  are wrapped with `%w`.

## Notes

- **Transactions (P6).** `WithTx` passes the two ports to `fn`; the deploy
  outcome path advances release status, archives the prior production release,
  and marks the deployment succeeded in one commit. `serve` still starts without
  a DB — only `tsugi migrate`/`release` open a pool (`TSUGI_DATABASE_URL`).
- **Shared Postgres, not a dedicated container.** Tsugi reuses the box's
  existing `postgres:16-alpine` (the same instance LazyScan runs), with its own
  `tsugi` database — no new container. One instance holds both environments'
  history; the `deployments.environment` column distinguishes them. See
  `docs/wiki/known-constraints.md`.
- **Migrations** live in `migrations/` (`golang-migrate` naming, no tool/lib
  dependency added) and are embedded into the binary. Apply with
  `tsugi migrate up` (`down` rolls back the last step). The runner is minimal —
  no dirty-state recovery or down-to-version.
- Not covered by a DB integration test (needs a live Postgres); validated by
  compile + `go vet`. The CLI orchestration is unit-tested in `internal/cli`
  with in-memory repos.
