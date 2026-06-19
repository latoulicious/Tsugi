# Resolutions

How findings were resolved. Paired with [`findings.md`](findings.md) — each entry
resolves a specific finding and must reference it; neither file is orphaned.

Format per resolution:

```md
## R-NNN <short title>  (resolves F-NNN)
- date:
- change: <what was done>
- files:
- verification: <how it was confirmed>
- constraints honored: <Do-Not rules respected>
```

## R-001 Guard `--target` against traversal  (resolves F-001)
- date: 2026-06-19
- change: Guard `--target` with `^[A-Za-z0-9_-]+$` right after arg parse and
  exit 2 on mismatch, before `TARGET` reaches any path or `source`. Blocks
  traversal/odd chars; valid targets unaffected. `--env` was already whitelisted.
- files: deploy/bin/deploy.sh (guard after the missing-arg check)
- verification: `bash -n` clean; `--target ../../x --env prod` → "invalid
  --target" exit 2; valid target unchanged; CodeRabbit re-run clean (0 findings).
- constraints honored: smallest safe change; no public-contract / behavior change
  for valid input; no unrelated cleanup; comment ≤2 lines.
