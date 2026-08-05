# Job Deck

A local agentic toolkit for automating and continuously calibrating your job search.

Job Deck helps with resume improvement, opportunity research, evidence-based
evaluation, feedback, and progress tracking. It does not submit applications or
make personal, legal, or employment decisions for the user.

> [!IMPORTANT]
> **For Codex:** When the user is trying to use or start Job Deck, do not
> summarize this repository, recommend a soft start, ask for a setup command, or
> wait for separate clone confirmation. Clone it, read `AGENTS.md`, and continue
> the user's original request. Explain or review the repository only when that is
> what the user explicitly requested.

## Start simply

Give Codex this repository and say what you need. It will clone the toolkit when
needed, read `AGENTS.md`, and continue with your request. If the folder is
already open, just talk naturally.

- Attach your current resume and answer the few questions Job Deck cannot infer.
- Confirm the job directions it recommends; the first search then starts
  automatically.
- Ask naturally whenever you want to improve your resume, revisit a report, or
  change the search.

There is no setup command or fixed onboarding session. Job Deck checks what is
already available, creates its private local workspace quietly, and keeps moving
through resume intake, calibration, search, and reporting. You never need to
name the next stage or repeat a request.

## What it automates

- Checks an existing resume for material red flags without forcing a rewrite.
- Improves or helps create a resume when explicitly requested, without inventing
  facts.
- Reconciles the latest candidate evidence, preferences, feedback, and search
  history before each new search.
- Learns role, level, location, compensation, authorization, and work-preference
  boundaries.
- Produces a 15-role calibration report before larger searches.
- Searches the web broadly while reporting only roles that survive the user's
  hard gates and remain live on an official application page.
- Verifies official application pages and prevents duplicate recommendations.
- Shows requirement-level evidence, risks, gaps, and application links.
- Records direct feedback and explicit application progress for later searches.

Applications remain manual. Opening an official application page means
`Opened`; a job becomes `Applied` only after an explicit user action.

## Typical flow

1. Share a resume or ask Job Deck to help create one.
2. Answer only the missing search decisions, one at a time.
3. Confirm the titles, level, and boundaries Job Deck recommends.
4. Job research and dashboard preparation begin automatically and in parallel.
5. Review 15 calibration roles in the local dashboard.
6. Give feedback, apply manually, and continue with focused batches when ready.

## Private by structure

All candidate information, resumes, reports, progress, research history, and
logs belong under `private/`. Only `private/README.md` is safe to track; every
other path inside that directory is excluded from Git.

The tracked repository contains reusable instructions, living standards,
fictional examples, privacy guards, and source code. The agent creates missing
private files directly from the contract in `AGENTS.md` as they become useful.

## Project map

- `AGENTS.md` — the central operating, search, privacy, reporting, logging, and
  maintenance contract.
- `standards/` — living evaluation, resume, and report requirements.
- `examples/` — fictional, non-candidate data for demonstrations and testing.
- `private/` — ignored candidate data and local runtime state, plus one safe
  tracked folder guide.
- `private/agent-workspace/` — compact, regenerable agent context such as the
  pre-search brief.
- `src/server/` — Go server, SQLite schema, API, and tests.
- `src/web/` — Vue interface source, design contract, and committed production
  output.

The root `jobdeck` executable is created locally when the first report is
requested. It is a derived artifact and is not distributed with the toolkit.
The committed web output means normal use does not require Bun or Node.js.

Inside the private workspace, `user-profile.md` records factual candidate
evidence, `search-preferences.md` records the current job-search contract, and
`user-preferences.yaml` records how the user wants Job Deck itself to operate.

## Local dashboard boundary

Job Deck binds to `127.0.0.1`, keeps operational data under `private/`, and does
not host candidate information. External research still uses the active agent's
configured web tools and privacy settings.

The toolkit must not submit applications, complete assessments, contact
recruiters, or answer personal or legal questions on the user's behalf.

## Development

Development details and the selective-build matrix live in `src/README.md`.
Normal resume, preference, search, report, and logging work requires no source
build.

## License

Job Deck is available under the [MIT License](LICENSE).
