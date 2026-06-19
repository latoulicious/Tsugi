# internal/release

P3 domain entity: a release and its lifecycle state machine. Pure domain —
stdlib only, no persistence (P5) or CLI (P6) yet.

## Symbols

- `Status` (string) — lifecycle state. Constants `StatusDraft`, `StatusCreated`,
  `StatusStaging`, `StatusProduction`, `StatusArchived`.
- `Status.Valid()` — reports whether the value is a known state.
- `Status.CanTransitionTo(target)` — reports whether the linear lifecycle allows
  `status → target`.
- `Release` (struct) — metadata `Version`, `CommitSHA`, `PreviousCommitSHA`,
  `CreatedAt` (exported, set once at construction) plus a guarded `status`.
- `New(version, commitSHA, previousCommitSHA, createdAt) (*Release, error)` —
  validates and starts the release at `StatusDraft`. `createdAt` is injected (no
  hidden clock) for testability.
- `Release.Status()` — current state (getter; `status` has no setter).
- `Release.TransitionTo(target) error` — the only mutator of `status`; enforces
  the transition table.
- Sentinel errors: `ErrEmptyVersion`, `ErrInvalidVersion`, `ErrEmptyCommit`,
  `ErrSameCommit`, `ErrInvalidStatus`, `ErrInvalidTransition`.

## Lifecycle

```txt
Draft → Created → Staging → Production → Archived
```

Strictly linear, taken verbatim from `PLAN.md`. Rollback (Production → previous
release) and abandon edges are **not** modelled here — they are P6 promotion/
rollback semantics, added when the CLI defines them.

## Notes

- `status` is unexported and mutated only via `TransitionTo`, so the state
  machine is the single path that changes lifecycle state. Plain metadata fields
  are exported and immutable by convention (set once in `New`).
- Version validation is a `v` prefix check only, not full semver; tighten when
  tags are parsed (P4 changelog / P6 CLI).
- `PreviousCommitSHA` may be empty (the first release has no predecessor); when
  set it must differ from `CommitSHA`.
- No repository port yet — the persistence interface arrives in P5 with the pgx
  implementation, to avoid a port with zero adapters.
