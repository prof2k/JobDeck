# Private workspace

This directory is the visible boundary for candidate-specific information and
locally generated state. This generic README is the only file here that may be
tracked. Everything else under `private/` must remain ignored.

- `user-profile.md` — verified resume and career evidence.
- `search-preferences.md` — desired roles, constraints, and calibration lessons.
- `user-preferences.yaml` — cost, parallelism, report, and interaction settings.
- `resume/original/` — the unchanged source resume.
- `reports/` — generated report artifacts and research manifests.
- `logs/` — append-only events and a compact readable activity log.
- `agent-workspace/` — compact derived context, including the pre-search brief.
- `runtime/` — local build receipts, process metadata, and temporary binaries.
- `toolchains/` — project-local development tools when the system lacks them.

The agent creates missing private files directly from `AGENTS.md`. Derived agent
notes are caches and never override the profile, preferences, resume, feedback,
or application history from which they were produced.
