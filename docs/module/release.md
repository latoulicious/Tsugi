# internal/release

P3 domain entity: a release and its lifecycle state machine. Pure domain —
stdlib only. P5 added a persistence-facing constructor + repository port (the
pgx adapter lives in [postgres.md](postgres.md)); the CLI is still P6.

## Symbols

- `Status` (string) — lifecycle state. Constants `StatusDraft`, `StatusCreated`,
  `StatusStaging`, `StatusProduction`, `StatusArchived`.
- `Status.Valid()` — reports whether the value is a known state.
- `Status.CanTransitionTo(target)` — reports whether the linear lifecycle allows
  `status → target`.
- `Release` (struct) — metadata `Version`, `CommitSHA`, `PreviousCommitSHA`,
  `Changelog`, `CreatedAt` (exported) plus a guarded `status`. `ID` (int64) is 0
  until persisted; the store assigns it via `RETURNING`.
- `New(version, commitSHA, previousCommitSHA, createdAt) (*Release, error)` —
  validates and starts the release at `StatusDraft`. `createdAt` is injected (no
  hidden clock) for testability.
- `Rehydrate(id, version, commitSHA, previousCommitSHA, changelog, status, createdAt) (*Release, error)` —
  reconstructs a persisted release for DB reads, setting `status` directly. Runs
  `New`'s field validation then checks `status.Valid()`.
- `Release.Status()` — current state (getter; `status` has no setter).
- `Release.TransitionTo(target) error` — the only mutator of `status`; enforces
  the transition table.
- `Repository` (interface) — `Create`, `GetByVersion`, `List`, `UpdateStatus`
  (P6). Consumer-defined port; implemented by `internal/postgres`.
- Sentinel errors: `ErrEmptyVersion`, `ErrInvalidVersion`, `ErrEmptyCommit`,
  `ErrSameCommit`, `ErrInvalidStatus`, `ErrInvalidTransition`, `ErrNotFound`.

## Lifecycle

```txt
Draft → Created → Staging → Production → Archived
                                            ↑ ──── ┘  (P6 rollback)
```

Linear from `PLAN.md`, plus the P6 `Archived → Production` edge: rollback
re-activates a previously archived release. Promotion archives the prior
production release, so only one release is in `Production` at a time.

## Notes

- `status` is unexported and mutated only via `TransitionTo`, so the state
  machine is the single path that changes lifecycle state for a live release.
  `Rehydrate` is the one exception — it sets `status` directly because a DB read
  must reconstruct a release at any persisted state without replaying the
  machine. It still has no public setter, so the guard holds for live objects.
- Version validation is a `v` prefix check only, not full semver; tighten when
  tags are parsed (P4 changelog / P6 CLI).
- `PreviousCommitSHA` may be empty (the first release has no predecessor); when
  set it must differ from `CommitSHA`.
- The `Repository` port arrived in P5 alongside its pgx adapter
  (`internal/postgres`); no port without an adapter.
