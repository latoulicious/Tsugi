# Tsugi Wiki

Project memory for Tsugi — release promotion and deployment orchestration
(次, "next": every release has a next step). Docs convention follows `../LazyScan`.

## Structure

| Path | Purpose |
|---|---|
| `architecture.md` | target architecture, VPS topology, phase table P1–P6 |
| `infra-plan.md` | project-local copy of `PLAN.md` + per-phase status |
| `known-constraints.md` | single-VPS facts, port map, branch invariant, risks |
| `running.md` | deploy a target's staging/prod, env table, tunnel setup |
| `findings.md` | code-review findings log (paired with `resolutions.md`) |
| `resolutions.md` | how findings were resolved (paired with `findings.md`) |
| `sessions/` | append-only session history (`DD-MM-YYYY.md`) |

## Related

- `../../../LazyScan/docs/wiki/` — docs/commit/comment convention source.
- `../../../LazyScan/docker-compose.yml` — first deploy target's compose
  (orchestrated, not forked).

> Doc rule: code is source of truth; note drift, keep notes concise, append
> session history — never overwrite it.
