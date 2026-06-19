# internal/postgres

P5 infrastructure adapter: pgx-backed implementation of the `release` and
`deployment` repository ports. First package with an external dependency
(`github.com/jackc/pgx/v5`); everything else stays stdlib.

## Symbols

- `Store` (struct) — aggregates the two repositories: `Releases`
  (`release.Repository`) and `Deployments` (`deployment.Repository`).
- `New(pool *pgxpool.Pool) *Store` — wires both sub-repos over one pool.
- `releaseRepo` / `deploymentRepo` (unexported) — concrete pgx impls. Split into
  two types because both ports declare `Create` (one struct can't carry both).

## Behavior

- Each call sets a `queryTimeout` (5s) via `context.WithTimeout`.
- `Create` inserts with `RETURNING id` and writes the new id back onto the
  entity (`rel.ID` / `dep.ID`).
- Reads go through the domain `Rehydrate` constructors, so a row that violates a
  domain invariant fails the scan rather than producing a bad entity.
- `pgx.ErrNoRows` maps to the domain `ErrNotFound` at the boundary; other errors
  are wrapped with `%w`.

## Notes

- **No transactions yet.** `Store` has no `WithTx`/`DBTX` — there is no atomic
  multi-write use case until P6 promotion. Marked with a `ponytail:` comment;
  add the tx seam when promotion needs to write release + deployment together.
- **Not wired into the running service.** `cmd/tsugi` still starts without a DB
  and serves `/version`/`/healthz` only. The pool + config (`TSUGI_DATABASE_URL`)
  and the migration runner land with the P6 CLI that consumes this store.
- **Shared Postgres, not a dedicated container.** Tsugi reuses the box's
  existing `postgres:16-alpine` (the same instance LazyScan runs), with its own
  `tsugi` database — no new container. One instance holds both environments'
  history; the `deployments.environment` column distinguishes them. See
  `docs/wiki/known-constraints.md`.
- **Migrations** live in `migrations/` (`golang-migrate` naming, no tool/lib
  dependency added). Apply manually for now:
  `migrate -path migrations -database "$TSUGI_DATABASE_URL" up`.
- Not covered by a DB integration test this phase (needs a live Postgres);
  validated by compile + `go vet`. The integration test lands when the pool is
  wired in P6.
