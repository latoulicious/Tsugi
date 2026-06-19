# internal/changelog

P4 changelog generation: turn conventional-commit subjects into grouped markdown
release notes. Pure — stdlib only, no git, no AI, no persistence.

## Symbols

- `Entry` (struct) — one parsed subject: `Type`, `Description`. Scope is parsed
  but discarded (the rendered output drops it, per `PLAN.md`).
- `Parse(subject) (Entry, bool)` — parse one conventional-commit subject
  (`type(scope)?!?: description`). `ok=false` for non-conventional lines; type is
  lowercased, an optional `(scope)` and breaking `!` are stripped.
- `Generate(subjects []string) string` — parse each subject, group by type, and
  render markdown. Non-conventional lines and unmapped types are skipped; empty
  input (or no mapped commits) yields `"_No notable changes._\n"`.

## Mapping

```txt
feat     -> ## Features
fix      -> ## Fixes
refactor -> ## Improvements
```

Only these three types render, in this order, taken verbatim from `PLAN.md`.
Sections with no entries are omitted. Other types (`chore`, `docs`, `test`, …)
are intentionally ignored — add a mapping here if a section is needed later.

## Notes

- `git log previous_sha..current_sha` is the intended source of `subjects`, but
  the git call is **not** here: `Generate` takes a plain `[]string` so it stays
  pure and unit-testable. The git exec lands with the P6 CLI (`release create`)
  that invokes it — same precedent as P3 deferring its repository adapter.
- The breaking marker `!` is tolerated so `feat!:`/`feat(x)!:` parse, but there
  is no `BREAKING` section — not in `PLAN.md`; defer until a release needs it.
- No persistence: the `releases.changelog` column is P5. P4 only produces the
  string.
