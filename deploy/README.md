# deploy/

Generic deployment layer. Tsugi orchestrates each target app's own compose; it
does not fork it.

```txt
bin/deploy.sh          interim deployer (Phase 1); replaced by the Go release CLI (Phase 6)
targets/<name>/        per-target, per-environment deploy config
  target.env           BASE_COMPOSE + per-env checkout paths (sourced by deploy.sh)
  docker-compose.<env>.override.yml   per-env port deltas (via the !override tag)
  cloudflared/config.yml.example      tunnel ingress: hostnames -> loopback ports
```

## How an env is separated

For `--env prod|staging`, `deploy.sh` combines these levers over the base compose:

1. **branch** — `prod`→`main`, `staging`→`development` (the core invariant).
2. **project** — `-p`: prod keeps the bare target name (`lazyscan`, matching the
   live stack), staging is suffixed (`lazyscan-staging`).
3. **override** — `docker-compose.<env>.override.yml` shifts published ports so
   both stacks coexist on one VPS.
4. **interpolation** — `--project-directory <checkout>` so compose reads the
   checkout's own `.env` for `${VAR}`. Tsugi does not inject an env-file; each
   env's secrets/domains live in its checkout (single source of truth).

## Adding a target

1. `mkdir targets/<name>`, write `target.env` (BASE_COMPOSE + checkout paths).
2. Add `docker-compose.prod.override.yml` / `docker-compose.staging.override.yml`
   for that app's port map.
3. Add a `cloudflared/config.yml.example`. Each checkout supplies its own `.env`
   for compose interpolation (Tsugi does not template it).

See [`../docs/wiki/running.md`](../docs/wiki/running.md) for the runbook.
