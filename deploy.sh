#!/usr/bin/env bash
# Deploys VN Stock Sim to Render from its Dockerfiles (render.yaml Blueprint,
# docs/guides/RENDER.md). Render builds the images itself, so "deploying" is:
# validate -> make sure the code is pushed -> trigger the services' deploys.
#
# Usage:
#   ./deploy.sh             # check, push current branch, trigger deploy hooks
#   ./deploy.sh --check     # only validate (docker build both images locally)
#   ./deploy.sh --no-push   # skip git push, just trigger deploy hooks
#
# Optional env (put in .env, gitignored): the "Deploy Hook" URLs from each
# Render service's Settings page. Without them, Render's auto-deploy on
# git push does the work.
#   RENDER_DEPLOY_HOOK_BACKEND=https://api.render.com/deploy/srv-...?key=...
#   RENDER_DEPLOY_HOOK_FRONTEND=https://api.render.com/deploy/srv-...?key=...
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

[ -f .env ] && { set -a; . ./.env; set +a; }

CHECK_ONLY=0; PUSH=1
for arg in "$@"; do
  case "$arg" in
    --check) CHECK_ONLY=1 ;;
    --no-push) PUSH=0 ;;
    -h|--help) sed -n 2,16p "$0"; exit 0 ;;
    *) echo "error: unknown option $arg" >&2; exit 1 ;;
  esac
done

[ -f render.yaml ] || { echo "error: render.yaml missing" >&2; exit 1; }

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  echo "==> Building images locally to catch Dockerfile errors early ..."
  docker build -q -t vss-backend-check ./backend
  docker build -q -t vss-frontend-check \
    --build-arg NEXT_PUBLIC_API_BASE_URL=http://vss-backend:8080 ./frontend
elif [ "$CHECK_ONLY" = 1 ]; then
  echo "error: docker is not available for --check" >&2; exit 1
else
  echo "warning: docker unavailable, skipping local build check" >&2
fi
[ "$CHECK_ONLY" = 1 ] && { echo "OK: images build."; exit 0; }

if [ "$PUSH" = 1 ]; then
  if [ -n "$(git status --porcelain)" ]; then
    echo "error: uncommitted changes -- commit them first so Render deploys what you expect." >&2
    exit 1
  fi
  branch=$(git rev-parse --abbrev-ref HEAD)
  echo "==> Pushing $branch (Render auto-deploys the linked branch) ..."
  git push -u origin "$branch"
fi

triggered=0
for name in BACKEND FRONTEND; do
  var="RENDER_DEPLOY_HOOK_$name"
  if [ -n "${!var:-}" ]; then
    echo "==> Triggering $name deploy hook ..."
    curl -fsS -X POST "${!var}" >/dev/null && echo "    started"
    triggered=1
  fi
done

if [ "$triggered" = 0 ]; then
  echo "No deploy hooks set; relying on Render auto-deploy from the push."
  echo "First time? In Render: New + > Blueprint > pick this repo (see docs/guides/RENDER.md)."
fi
echo "Done. Watch progress at https://dashboard.render.com"
