# Tsugi

> 次 — "next". Every release has a next step. Tsugi manages that step.

Release promotion and deployment orchestration for personal infrastructure —
release visibility, changelogs, promotion, and rollback on a single VPS.

It answers: what version is running, what commit is deployed, what changed, when
it deployed, and can it be rolled back.

## Status

Phase 1 — **Environment Separation** (infra scaffold). Production deploys
`main`, staging deploys `dev`, on distinct containers and domains. No Go code
yet; the Go service + `release` CLI land in Phases 2–6.

See [`docs/wiki/`](docs/wiki/README.md) for architecture, the phased plan, and
the deploy runbook.

## Layout

```txt
docs/wiki/        project memory (architecture, infra-plan, runbook, sessions)
deploy/           generic deploy layer
  targets/        per-target, per-environment deploy config (first: lazyscan)
  bin/deploy.sh   interim deployer (superseded by the Go release CLI in Phase 6)
```

## Quick start (operator)

```sh
deploy/bin/deploy.sh --target lazyscan --env staging   # dev → staging
deploy/bin/deploy.sh --target lazyscan --env prod      # main → production
```

Fill `deploy/targets/<target>/.env.<env>` from the `.example` files first. See
[`docs/wiki/running.md`](docs/wiki/running.md).
