# Job Deck source

This folder is relevant only when the toolkit itself changes. Normal resume,
research, report, feedback, and application-tracking work requires no source
build.

## Layout

- `server/` — Go HTTP service, SQLite state, import validation, activity logging,
  and embedded production UI.
- `web/` — Vue source, its design contract, and the committed `dist/` consumed
  by the server build.
- `scripts/` — narrow build and privacy helpers.

The fictional import fixture lives at `examples/fictional-report.json`; it
contains no candidate information.

## Build only the affected layer

| Changed files | Command path |
|---|---|
| `src/server/**` | Go test, copy current web `dist`, rebuild Go |
| `src/web/src/**`, `src/web/public/**`, or web config | Bun typecheck/build, then rebuild Go |
| `private/**`, `standards/**`, `examples/**`, or root Markdown | No build |

Never run the web build as a precaution. `src/web/dist/` is committed so a fresh
user can compile the server without Bun or Node.js.

## Server-only build

From the repository root:

```bash
./src/scripts/build-server.sh
```

The script copies the existing `src/web/dist/` into the server embed directory,
runs Go tests, builds a temporary executable, and replaces the root executable
only after success.

## UI change

Only after changing the UI:

```bash
cd src/web
bun install --frozen-lockfile
bun run typecheck
bun run build
cd ../..
./src/scripts/build-server.sh
```

Do not commit `node_modules`. Do commit `src/web/dist/` and `bun.lock` so setup
does not require a frontend toolchain.

## Runtime paths

All generated runtime files belong under `private/runtime/`. Candidate state
lives in `private/jobdeck.sqlite`; agent-readable artifacts and logs remain in
their corresponding private folders.
