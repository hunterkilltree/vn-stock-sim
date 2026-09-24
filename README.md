# VN Stock Sim

A simulation-first platform for analyzing, practicing, and backtesting trades on Vietnamese stocks (HOSE, HNX, UPCOM) without real money — *"Trade the past before you trade the future."*

- `backend/` — Go + Gin API (`/api/v1`), Postman collection in `backend/postman/`
- `frontend/` — Next.js app
- `design/` — design export used as the literal visual spec

## Documentation

| Folder | What's in it |
|---|---|
| [docs/product/](docs/product/) | Product summary and the V1→V4 feature roadmap |
| [docs/architecture/](docs/architecture/) | API spec, charting-library integration, architecture diagram |
| [docs/guides/](docs/guides/) | [Local dev](docs/guides/RUNNING.md) and [Docker demo](docs/guides/DOCKER.md) |
| [docs/roadmap/](docs/roadmap/) | [Full implementation plan](docs/roadmap/FULL-APP-PLAN.md) and [current status](docs/roadmap/RESUME.md) |
| [docs/roadmap/phases/](docs/roadmap/phases/) | Planning + verification record for each phase |
| [tools/excalidraw-diagram/](tools/excalidraw-diagram/) | Excalidraw diagram-generation skill (upstream tooling) |

Quick start: `./run.sh` (Docker), or see [docs/guides/RUNNING.md](docs/guides/RUNNING.md) for hot-reload dev.
