# Project Structure

## Source-controlled directories
- `backend/` — Go backend source code and tests
- `frontend/` — React frontend source code and tests
- `docs/` — design, planning, and maintenance documentation
- `deploy/` — deployment configuration tracked in git when it is source, not packaged output

## Runtime directories
- `uploads/` — uploaded files generated at runtime
- `backups/` — backup archives generated at runtime
- `logs/` — runtime logs

These directories are environment data, not source code. They should be created by the runtime environment or deployment scripts and should not be committed.

## Local-only workspace directories
- `.planning/`
- `.worktrees/`
- `.agents/`
- `.claude/`

These directories store local planning, agent, and tooling state.

## Build output
- `frontend/dist/`
- `backend/*.exe`
- `backend/main`

Build output can be deleted and regenerated.
