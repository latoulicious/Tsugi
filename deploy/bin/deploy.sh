#!/usr/bin/env bash
# Tsugi interim deployer (Phase 1) — env separation: prod←main, staging←dev.
# Superseded by the Go `release` CLI in Phase 6.
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: deploy.sh --target <name> --env <prod|staging> [--ref <sha>]

Resolves branch (prod->main, staging->dev), compose project, override, and
env-file, then runs `docker compose ... up -d` against the target checkout.
With --ref, checks out that commit (detached) instead of the branch HEAD —
used by `tsugi release rollback` to redeploy a previous release.

Targets live in deploy/targets/<name>/ with a target.env config.
EOF
}

TARGET=""
ENV=""
REF=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --target) TARGET="${2:-}"; shift 2 ;;
    --env)    ENV="${2:-}"; shift 2 ;;
    --ref)    REF="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown arg: $1" >&2; usage; exit 2 ;;
  esac
done

[[ -n "$TARGET" && -n "$ENV" ]] || { echo "missing --target/--env" >&2; usage; exit 2; }

# Guard the path/source input: no traversal, no odd chars.
[[ "$TARGET" =~ ^[A-Za-z0-9_-]+$ ]] || { echo "invalid --target: $TARGET" >&2; exit 2; }

# Guard --ref: a git SHA only (no branch names or options into checkout).
[[ -z "$REF" || "$REF" =~ ^[0-9a-fA-F]{7,40}$ ]] || { echo "invalid --ref: $REF" >&2; exit 2; }

# env -> branch + checkout (the prod←main / staging←dev invariant).
case "$ENV" in
  prod)    BRANCH="main" ;;
  staging) BRANCH="dev" ;;
  *) echo "env must be prod|staging" >&2; exit 2 ;;
esac

DEPLOY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET_DIR="$DEPLOY_DIR/targets/$TARGET"
[[ -d "$TARGET_DIR" ]] || { echo "no such target: $TARGET ($TARGET_DIR)" >&2; exit 2; }

# Per-target config: BASE_COMPOSE (relative to checkout), CHECKOUT_PROD/STAGING.
# shellcheck source=/dev/null
source "$TARGET_DIR/target.env"

BASE_COMPOSE="${BASE_COMPOSE:-docker-compose.prod.yml}"
case "$ENV" in
  prod)    CHECKOUT="${CHECKOUT_PROD:-}" ;;
  staging) CHECKOUT="${CHECKOUT_STAGING:-}" ;;
esac
[[ -n "$CHECKOUT" ]] || { echo "checkout path unset in target.env for $ENV" >&2; exit 2; }
[[ -d "$CHECKOUT" ]] || { echo "checkout missing: $CHECKOUT" >&2; exit 2; }

ENV_FILE="$TARGET_DIR/.env.$ENV"
[[ -f "$ENV_FILE" ]] || { echo "missing $ENV_FILE (copy from .env.$ENV.example)" >&2; exit 2; }

OVERRIDE="$TARGET_DIR/docker-compose.$ENV.override.yml"
PROJECT="$TARGET-$ENV"

echo "==> $PROJECT  branch=$BRANCH  ref=${REF:-HEAD}  checkout=$CHECKOUT"
git -C "$CHECKOUT" fetch --prune origin
if [[ -n "$REF" ]]; then
  git -C "$CHECKOUT" checkout --detach "$REF"
else
  git -C "$CHECKOUT" checkout "$BRANCH"
  git -C "$CHECKOUT" pull --ff-only origin "$BRANCH"
fi

COMPOSE_ARGS=(-p "$PROJECT" -f "$CHECKOUT/$BASE_COMPOSE")
[[ -f "$OVERRIDE" ]] && COMPOSE_ARGS+=(-f "$OVERRIDE")

docker compose "${COMPOSE_ARGS[@]}" --env-file "$ENV_FILE" up -d --build --remove-orphans
echo "==> $PROJECT deployed"
