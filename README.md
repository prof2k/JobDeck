# Job Deck

A local agentic toolkit for automating and continuously calibrating your job search.

Job Deck helps with resume improvement, opportunity research, evidence-based
evaluation, feedback, and progress tracking. It does not submit applications or
make personal, legal, or employment decisions for the user.

## Start simply

Open this folder with a capable agent. The agent reads `AGENTS.md`, which explains
the entire workflow.

- First time: say `/setup` and provide a resume, or ask for help creating one.
- Whenever you want a researched batch: say **“Find me jobs.”**
- To continue reviewing existing results: say **“Show my report.”**
- To revisit the resume: say **“Improve my resume.”**

If you give an agent the repository URL before it exists locally, the agent
should briefly explain Job Deck and ask permission to clone it. Nothing should
be cloned until you say yes.

## What it automates

- Reviews or helps create a resume without inventing facts.
- Reconciles the latest candidate evidence, preferences, feedback, and search
  history before each new search.
- Learns role, level, location, compensation, authorization, and work-preference
  boundaries.
- Produces a 15-role calibration report before larger searches.
- Searches broadly while reporting only roles that survive the user's hard
  gates.
- Verifies official application pages and prevents duplicate recommendations.
- Shows requirement-level evidence, risks, gaps, and application links.
- Records direct feedback and explicit application progress for later searches.

Applications remain manual. Opening an official application page means
`Opened`; a job becomes `Applied` only after an explicit user action.

## Typical flow

1. Complete `/setup` once.
2. Say “Find me jobs.”
3. Review 15 calibration roles in the local Job Deck dashboard.
4. Give feedback so the search contract can improve.
5. Apply manually through official employer pages.
6. Continue with focused batches, normally 30 verified roles at a time.

## Private by structure

All candidate information, resumes, reports, progress, research history, and
logs belong under `private/`. Only `private/README.md` is safe to track; every
other path inside that directory is excluded from Git.

The tracked repository contains reusable instructions, living standards,
fictional examples, privacy guards, and source code. During setup, the agent
creates missing private files directly from the contract in `AGENTS.md`.

## Project map

- `AGENTS.md` — the central setup, search, privacy, reporting, logging, and
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
The committed web output means normal setup does not require Bun or Node.js.

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
