# Deploying to Render

```bash
./deploy.sh
```

Render builds the Docker images itself from `render.yaml` (Postgres,
`vss-backend` private service, `vss-frontend` public web service). The script
build-checks both images locally (if Docker is running), pushes the current
branch, and calls the optional deploy hooks.

## One-time setup

1. Render dashboard -> **New + -> Blueprint**, pick this repo and branch. It
   creates all three resources from `render.yaml` (`JWT_SECRET` is generated).
2. Optional: copy each service's *Deploy Hook* URL into `.env` as
   `RENDER_DEPLOY_HOOK_BACKEND` / `RENDER_DEPLOY_HOOK_FRONTEND`.

After that, every `./deploy.sh` ships the latest commit. Flags: `--check`
(local build only), `--no-push` (only trigger hooks).

## Notes

- The backend is a private service (not on the free tier); only the frontend
  is public. The frontend reaches it at `http://vss-backend:8080`, baked in at
  build time as `NEXT_PUBLIC_API_BASE_URL`.
- The free Postgres plan expires after 30 days; upgrade `vss-db` for real use.
- `gemini-bridge` (Trợ lý Quant's optional provider) is not in the Blueprint.
- Not verified against a live Render account.
