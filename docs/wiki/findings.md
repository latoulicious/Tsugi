# Findings

Code-review findings log. Paired with [`resolutions.md`](resolutions.md) — every
finding that gets fixed must have a matching resolution entry; the two are
interconnected and must not be orphaned.

Format per finding:

```md
## F-NNN <short title>
- date:
- source: <review tool / PR / manual>
- severity: low | medium | high
- location: path:line
- problem:
- status: open | resolved (→ R-NNN)
```

## F-001 Unvalidated `--target` flows into path + `source`
- date: 2026-06-19
- source: CodeRabbit CLI (`coderabbit review --agent`), Phase 1 scaffold
- severity: medium (reported major)
- location: deploy/bin/deploy.sh:38,43
- problem: `TARGET` from `--target` is concatenated into
  `TARGET_DIR="$DEPLOY_DIR/targets/$TARGET"` and then `source`d. A traversal
  value (e.g. `../../x`) could read/source a file outside `targets/`.
- status: resolved (→ R-001)
