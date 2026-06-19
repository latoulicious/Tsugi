# Running Tsugi

Phase 1 deploy runbook. `deploy/bin/deploy.sh` is the interim deployer; the Go
`release` CLI replaces it in Phase 6. Run it on the VPS — it operates on local
checkouts and the local Docker daemon.

## Deploy

```sh
deploy/bin/deploy.sh --target lazyscan --env staging   # dev → staging stack
deploy/bin/deploy.sh --target lazyscan --env prod      # main → production stack
deploy/bin/deploy.sh --help
```

What it does, per env:

1. resolve branch (`prod`→`main`, `staging`→`dev`) and project
   (`<target>-<env>`).
2. `cd` the env checkout, `git fetch` + `checkout` + `pull --ff-only` the branch.
3. `docker compose -p <project> -f <base> -f <override> --env-file <env-file>
   up -d --build --remove-orphans`.

## Per-target config

`deploy/targets/<target>/target.env` (sourced by the script):

| Variable | Meaning |
|---|---|
| `BASE_COMPOSE` | base compose path **relative to the checkout** (default `docker-compose.prod.yml`) |
| `CHECKOUT_PROD` | VPS path of the `main` checkout (e.g. `/opt/lazyscan-prod`) |
| `CHECKOUT_STAGING` | VPS path of the `dev` checkout (e.g. `/opt/lazyscan-staging`) |

## Environment files

Copy the templates and fill real values (git-ignored):

```sh
cp deploy/targets/lazyscan/.env.prod.example    deploy/targets/lazyscan/.env.prod
cp deploy/targets/lazyscan/.env.staging.example deploy/targets/lazyscan/.env.staging
```

These hold compose interpolation vars only (domains, infra creds). The target
api's own secrets stay in `LazyScan/api/.env` — do not duplicate them here.

## Port map

| Service | Production | Staging |
|---|---|---|
| app edge | `127.0.0.1:8080` | `127.0.0.1:8081` |
| minio S3 API | `127.0.0.1:9000` | `127.0.0.1:9100` |
| minio console | `127.0.0.1:9001` | `127.0.0.1:9101` |

## Cloudflare Tunnel

Copy `deploy/targets/lazyscan/cloudflared/config.yml.example` to `config.yml`,
fill the tunnel id/domain, then route the hostnames:

```sh
cloudflared tunnel route dns <tunnel> api.<domain>
cloudflared tunnel route dns <tunnel> staging-api.<domain>
cloudflared tunnel route dns <tunnel> s3.<domain>
cloudflared tunnel route dns <tunnel> staging-s3.<domain>
cloudflared tunnel run <tunnel>
```

## First checkout on the VPS

```sh
git clone <lazyscan-stack-repo> /opt/lazyscan-prod    && git -C /opt/lazyscan-prod    checkout main
git clone <lazyscan-stack-repo> /opt/lazyscan-staging && git -C /opt/lazyscan-staging checkout dev
```

## Local validation (no VPS, no deploy)

```sh
# Merge + port check (prod 8080, staging 8081):
docker compose -p lazyscan-prod \
  -f ../LazyScan-Stack/docker-compose.prod.yml \
  -f deploy/targets/lazyscan/docker-compose.prod.override.yml \
  --env-file deploy/targets/lazyscan/.env.prod.example config | grep -A2 published

cloudflared tunnel ingress validate deploy/targets/lazyscan/cloudflared/config.yml.example
bash -n deploy/bin/deploy.sh
```

## Acceptance (Phase 1 success criterion)

- [ ] `docker compose -p lazyscan-prod ... config` → app edge on `:8080`.
- [ ] `docker compose -p lazyscan-staging ... config` → app edge on `:8081`.
- [ ] `api.<domain>` serves production (built from `main`).
- [ ] `staging-api.<domain>` serves staging (built from `dev`).
- [ ] **Production is no longer running `dev`.**
