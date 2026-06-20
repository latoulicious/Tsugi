# Running Tsugi

Phase 1 deploy runbook. `deploy/bin/deploy.sh` is the interim deployer; the Go
`release` CLI replaces it in Phase 6. Run it on the VPS — it operates on local
checkouts and the local Docker daemon.

## Tsugi service (Phase 2)

The Go service exposes `GET /version`. Build stamps version/commit/date from
git via ldflags; defaults are `dev`/`none`/`unknown`.

```sh
make build && ./bin/tsugi      # listens on :8080 (override TSUGI_ADDR)
make image                     # docker image, same metadata via --build-arg
make test && make vet

curl -s localhost:8080/version # {"version":"v1.2.0","commit":"abc123","deployed_at":"...Z"}
curl -s localhost:8080/healthz # {"status":"ok","service":"tsugi"}
```

`deployed_at` is build time (rebuild-on-deploy ⇒ build ≈ deploy). The service's
own compose / VPS deploy wiring is not yet built (P5/P6). Package docs:
`docs/module/`.

## Release CLI (Phase 6)

The `tsugi` binary drives releases against the `tsugi` Postgres database. All
CLI paths need `TSUGI_DATABASE_URL` (`serve` does not). `TSUGI_TARGET` defaults
to `lazyscan`; `TSUGI_DEPLOY_DIR` to `deploy`.

```sh
export TSUGI_DATABASE_URL="postgres://tsugi:...@127.0.0.1:5432/tsugi"

tsugi migrate up                 # apply embedded migrations (down = last step)

tsugi release create v1.2.0      # snapshot staging commit + changelog -> Staging
tsugi release list               # version / status / commit / created
tsugi release show v1.2.0        # detail + changelog
tsugi release promote v1.2.0     # staging -> production (real prod deploy)
tsugi release rollback v1.1.5    # redeploy an archived release (deploy.sh --ref)
```

`promote`/`rollback` run the real `docker compose` deploy on the box, record a
`deployments` row, and only advance release status when the deploy succeeds.
Rollback redeploys a specific commit via `deploy.sh --ref <sha>`.

## Deploy (interim script)

`deploy.sh` is still callable directly (and is what the CLI shells out to):

```sh
deploy/bin/deploy.sh --target lazyscan --env staging   # development → staging stack
deploy/bin/deploy.sh --target lazyscan --env prod      # main → production stack
deploy/bin/deploy.sh --target lazyscan --env prod --ref <sha>  # rollback a commit
deploy/bin/deploy.sh --help
```

What it does, per env:

1. resolve branch (`prod`→`main`, `staging`→`development`) and project
   (prod = `lazyscan`, staging = `lazyscan-staging`).
2. `cd` the env checkout, `git fetch` + `checkout` + `pull --ff-only` the branch.
3. `docker compose -p <project> --project-directory <checkout> -f <base>
   -f <override> up -d --build --remove-orphans` — interpolation from the
   checkout's own `.env`.

## Per-target config

`deploy/targets/<target>/target.env` (sourced by the script):

| Variable | Meaning |
|---|---|
| `BASE_COMPOSE` | base compose path **relative to the checkout** (default `docker-compose.yml`) |
| `CHECKOUT_PROD` | VPS path of the `main` checkout (e.g. `/opt/lazyscan-prod`) |
| `CHECKOUT_STAGING` | VPS path of the `development` checkout (e.g. `/opt/lazyscan-staging`) |

## Environment files

Tsugi does not template env-files. Each checkout supplies its own `.env` in the
checkout root (git-ignored); `docker-compose.yml` interpolates it via
`--project-directory`. The prod checkout holds prod's `.env`, the staging
checkout holds staging's (own domain, own R2 creds) — single source of truth, no
duplication in this repo. See `../LazyScan/.env.example` for the full variable set
(`JWT_SECRET`, `STORAGE_*` R2, `SMTP_*`, `DEFAULT_SUPERUSER_*`, …).

## Port map

| Service | Production | Staging |
|---|---|---|
| web (nginx edge) | `127.0.0.1:8081` | `127.0.0.1:8082` |

No minio — uploads go to R2. On the box, `8080` is taken by dozzle and `8081` by
the live prod stack, so staging publishes on `8082`.

## Cloudflare Tunnel

The VPS runs one shared tunnel (`vps`). Add the app hostnames to its ingress
(`/etc/cloudflared/config.yml`, see the example) above the `404` catch-all, then
route DNS:

```sh
cloudflared tunnel route dns vps lazyscan.my.id
cloudflared tunnel route dns vps staging.lazyscan.my.id
```

## First checkout on the VPS

```sh
git clone <lazyscan-repo> /opt/lazyscan-prod    && git -C /opt/lazyscan-prod    checkout main
git clone <lazyscan-repo> /opt/lazyscan-staging && git -C /opt/lazyscan-staging checkout development
```

Each checkout then needs its own `.env` (copy from the repo's `.env.example`, fill
prod/staging values) — compose reads it for interpolation.

## Local validation (no VPS, no deploy)

```sh
# Merge + port check (prod 8081, staging 8082); needs ../LazyScan/.env present:
docker compose -p lazyscan \
  --project-directory ../LazyScan \
  -f ../LazyScan/docker-compose.yml \
  -f deploy/targets/lazyscan/docker-compose.prod.override.yml \
  config | grep -A2 published

cloudflared tunnel ingress validate deploy/targets/lazyscan/cloudflared/config.yml.example
bash -n deploy/bin/deploy.sh
```

## Acceptance (Phase 1 success criterion)

- [ ] `docker compose -p lazyscan ... config` → web edge on `:8081`.
- [ ] `docker compose -p lazyscan-staging ... config` → web edge on `:8082`.
- [ ] `lazyscan.my.id` serves production (built from `main`).
- [ ] `staging.lazyscan.my.id` serves staging (built from `development`).
- [ ] **Production is no longer running `development`.**
