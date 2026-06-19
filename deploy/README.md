# deploy/

Generic deployment layer. Tsugi orchestrates each target app's own compose; it
does not fork it.

```txt
bin/deploy.sh          interim deployer (Phase 1); replaced by the Go release CLI (Phase 6)
targets/<name>/        per-target, per-environment deploy config
  target.env           BASE_COMPOSE + per-env checkout paths (sourced by deploy.sh)
  docker-compose.<env>.override.yml   per-env port deltas (via the !override tag)
  .env.<env>.example   compose interpolation vars (copy to .env.<env>, git-ignored)
  cloudflared/config.yml.example      tunnel ingress: hostnames -> loopback ports
```

## How an env is separated

For `--env prod|staging`, `deploy.sh` combines four levers over the base compose:

1. **branch** — `prod`→`main`, `staging`→`dev` (the core invariant).
2. **project** — `-p <target>-<env>` → a distinct container set.
3. **override** — `docker-compose.<env>.override.yml` shifts published ports so
   both stacks coexist on one VPS.
4. **env-file** — `.env.<env>` supplies the env's domains and infra creds.

## Adding a target

1. `mkdir targets/<name>`, write `target.env` (BASE_COMPOSE + checkout paths).
2. Add `docker-compose.prod.override.yml` / `docker-compose.staging.override.yml`
   for that app's port map.
3. Add `.env.prod.example` / `.env.staging.example` and a `cloudflared/
   config.yml.example`.

See [`../docs/wiki/running.md`](../docs/wiki/running.md) for the runbook.
