# internal/git

P6 thin `os/exec` wrapper over the `git` binary — the changelog input source
that P4 deferred. No library dependency.

## Symbols

- `HeadSHA(ctx, dir)` — `git -C dir rev-parse HEAD`.
- `Subjects(ctx, dir, prev, head)` — `git -C dir log --format=%s prev..head`
  (or up to `head` when `prev` is empty); returns subjects newest-first for
  `changelog.Generate`.
- `Default` (struct) — adapts the package functions to `cli.GitReader`.

## Notes

- Reads from the target's checkout directory (e.g. the LazyScan staging
  checkout), not the Tsugi repo.
- Output-only; never mutates a checkout. Branch/commit checkout for deploys is
  `deploy.sh`'s job, not this package's.
