---
last_updated: 2026-08-05
status: living
---

# Job Deck operating contract

Job Deck is a local toolkit for automating and continuously calibrating a
person's job search. The agent performs the research and reasoning; the local
dashboard and its reports are the durable output surface.

During normal use, this is a job-search workspace rather than a general coding
repository. Do not apply unrelated product-development conventions, broad
coding skills, or infrastructure workflows to operational job-search work. Use
only capabilities essential to resume inspection, document creation, web
research, job evidence, local reporting, or an explicit source-maintenance
request.

## Intent routing

Interpret these as natural-language workflow instructions even if the current
agent does not implement native slash commands:

- `/setup` — perform first-time onboarding only.
- `Find me jobs` — reconcile current search state, research a new batch, and
  produce a report.
- `Improve my resume` — inspect the maintained resume and propose or produce
  truthful revisions under the resume standard.
- `Show my report` — start the current local runtime when necessary and open the
  existing report without beginning a new search.

If a search is requested before onboarding is complete, finish only the missing
material setup first. Ask only questions whose answers could change eligibility,
search scope, evaluation, or what the agent may truthfully claim.

## Repository entry and clone consent

When Job Deck is already open locally, read this file and continue without
asking the user to reconfirm the repository.

When a user supplies a Job Deck repository URL from outside a checkout:

1. Explain in one sentence that it is a local job-search toolkit that stores
   candidate data outside tracked Git files.
2. Ask whether the user wants the repository cloned locally.
3. Clone and enter it only after an explicit yes.
4. If the user declines, stop without creating files or changing external
   state.

Never create a remote repository, add a remote, commit, push, publish, or deploy
unless the user explicitly asks.

## Source of truth and required reading

Repository memory and `private/agent-workspace/` are caches. Candidate-specific
files and state under `private/` are authoritative.

Read this file first. Then read only the standards relevant to the work:

- Before evaluating, ranking, or excluding jobs, read
  `standards/evaluation.md`.
- Before reviewing or changing a resume, read `standards/resume.md`.
- Before creating or changing report content or behavior, read
  `standards/report.md`.
- Before changing source, read `src/README.md`; for interface work also read
  `src/web/README.md`.

Resolve conflicting guidance in this order:

1. The user's current explicit correction.
2. Confirmed facts and preferences under `private/`.
3. This operating contract and the relevant living standard.
4. Derived agent notes such as `private/agent-workspace/search-brief.md`.

## Living standards and update discipline

`AGENTS.md` and the files under `standards/` are living operating documents.
Each carries `last_updated` metadata. Update that date whenever a substantive
rule changes, not merely when wording or formatting changes.

- Candidate-specific facts, dislikes, corrections, and calibration lessons stay
  under `private/`.
- Promote a lesson into a public standard only when it is reusable across
  candidates, contains no personal information, and materially improves the
  workflow.
- When a public standard changes, update its `last_updated` field and keep this
  file's routing consistent with it.
- Never let a compact agent summary silently override its cited source files.
- Record meaningful candidate-state changes in the private activity log.

## Privacy invariant

Every user-specific detail belongs under `private/`, including resumes, names,
contact details, career facts, preferences, reports, research manifests,
feedback, application progress, logs, toolchains, databases, and generated
executables.

`private/README.md` is the only safe tracked file inside that directory. It is a
generic folder guide and must never contain candidate data.

- Never place personal data in `README.md`, `AGENTS.md`, `standards/`, `examples/`,
  `src/`, tests, commit messages, or other tracked paths.
- Before a commit or push, confirm that `git ls-files private` contains only
  `private/README.md`, then run `src/scripts/privacy-check.sh`.
- Do not use `git add -f` on ignored content under `private/`.
- Do not disable or remove Git. Git protects and updates the reusable toolkit;
  the ignored private boundary protects the candidate.

## Private workspace bootstrap

The tracked `private/README.md` makes the private boundary visible in a fresh
clone. When `/setup` finds missing private state, create it directly; there is no
second template tree.

Create these ignored paths as needed:

- `private/resume/original/`
- `private/reports/`
- `private/logs/events.jsonl`
- `private/logs/log.md`
- `private/agent-workspace/`
- `private/runtime/`
- `private/user-profile.md`
- `private/search-preferences.md`
- `private/user-preferences.yaml`

Create `private/toolchains/` only when a project-local toolchain is actually
needed. The `runtime/` folder holds device-specific build receipts, process
metadata, and temporary executable state; it is not a second source of candidate
truth.

Use dated headings in the Markdown state files. Use `Last updated` for the user
profile and search preferences, and `Last consolidated` for the readable log.
Initialize operational preferences with this shallow YAML schema:

```yaml
version: 1
last_updated: "YYYY-MM-DD"
last_confirmed: null
research_mode: balanced
ai_cost_sensitivity: standard
parallel_workers: dynamic
prefer_batched_discovery: true
stop_after_enough_verified_roles: true
calibration_batch_size: 15
later_batch_size: 30
open_dashboard_when_report_ready: true
keep_last_report_offline: true
ask_only_material_questions: true
explain_technical_setup_only_when_needed: true
log_detail: compact
```

Do not copy the fictional example report into a candidate profile, history, or
database unless the user explicitly asks to load the demo.

## `/setup`: onboarding only

Do not build or start Job Deck during onboarding.

1. Create the missing private workspace described above.
2. Ask the user to provide their resume. Preserve the original unchanged in
   `private/resume/original/`.
3. If the user has no resume, build one through evidence-based questions.
4. Review the resume for unclear claims, contradictions, red flags, and ATS
   issues. Never invent technologies, metrics, dates, employment, education,
   responsibilities, or impact.
5. Ask only the remaining high-impact questions about role families, titles,
   level, location, work arrangement, relocation, compensation, employment type,
   authorization, sponsorship, industries, preferences, and dealbreakers.
6. Write confirmed evidence to `private/user-profile.md`, the search contract to
   `private/search-preferences.md`, and operational choices to
   `private/user-preferences.yaml`.
7. Record a compact setup event in `private/logs/` and stop. Wait until the user
   says `Find me jobs` or otherwise requests a report before preparing the
   runtime.

## Required pre-search reconciliation

Before every external job search, the coordinating agent must reconcile local
state. Do not browse for new roles and do not start research workers until this
preflight is complete.

Review:

1. `private/user-profile.md`, including its last update and evidence boundaries.
2. `private/search-preferences.md`, including its last update, hard gates,
   calibration lessons, and unresolved questions.
3. `private/user-preferences.yaml`, including `last_updated`, research mode,
   cost sensitivity, worker preference, and batch size.
4. The maintained resume's identity and update time. Re-read the full resume
   when it changed or when the evidence inventory is incomplete.
5. Pending dashboard feedback and any unresolved user corrections.
6. Previous report manifests, retained canonical identities, and SQLite history
   needed to prevent duplicates.
7. The recent compact activity log and existing agent-workspace notes.

Then regenerate `private/agent-workspace/search-brief.md`. It is a derived cache,
not a source of truth, and should be replaced rather than allowed to grow
indefinitely. Include:

- generation time and the source files and update dates reviewed;
- the current search goal and requested batch type;
- target, adjacent, and excluded role families;
- eligibility constraints, hard gates, dealbreakers, and soft preferences;
- verified evidence, evidence labels, known gaps, and prohibited claims;
- previous search coverage, duplicate boundaries, and unresolved feedback;
- research mode, cost limit, worker plan, batch size, and stopping condition;
- material open questions and the planned source partitions.

When sources conflict, repair the underlying private files first and regenerate
the brief. Ask one concise question only when the conflict is material to search
scope, eligibility, or truthful evaluation. Ordinary stale formatting or dates
should be reconciled locally without interrupting the user.

Parallel workers receive the same compact brief plus one distinct source, role
family, or geography assignment. They do not receive the full raw history unless
their assignment requires it. Centralize hard gates, canonical deduplication,
evidence review, and ranking in the coordinating agent.

## First report and lazy runtime

When the user requests the first report, begin runtime preparation and job
research concurrently. Runtime failure must never block delivery of report
artifacts.

- If `jobdeck` is current, start it.
- Otherwise, use an existing compatible Go toolchain or place a verified
  official Go archive under `private/toolchains/go/`. Do not require the user to
  install Go, modify PATH, use an administrator account, or perform technical
  steps.
- Verify the official archive checksum before use.
- Build the committed web output into the Go executable without running Bun.
- Store the build receipt under `private/runtime/` and create the derived
  executable at the project root.
- Bind only to `127.0.0.1`. If the fixed port is occupied by another application,
  report the conflict rather than choosing an unpredictable port.

## Cost-aware research orchestration

Use internal parallel workers only for independent work that reduces elapsed
time or increases useful source coverage. Keep one coordinating agent responsible
for the reconciled search brief, hard gates, canonical deduplication, and final
ranking.

Read `private/user-preferences.yaml` and default to `balanced` unless the user has
expressed another preference:

- `efficient`: one worker or the minimum useful fan-out; batch queries, apply
  hard gates early, avoid redundant validation, and stop after enough verified
  roles survive.
- `balanced`: a small number of workers divided by independent source, role
  family, or geography, followed by centralized validation.
- `thorough`: broader independent coverage and additional cross-checking when
  the user values recall over cost.

Never parallelize identical searches. Never create user-visible agent threads
unless the user asks. Reduce duplicated work before reducing evidence quality.
If the user sets an AI-cost or system-resource limit, obey it and record the
preference in `private/user-preferences.yaml`.

## Search and evaluation contract

- Use 15 verified roles for the first calibration batch.
- Default later batches to 30 unless the user chooses another size.
- The first batch should test plausible preference boundaries without including
  obviously ineligible jobs merely for variety.
- Search effort may cover hundreds of candidates; output only roles that survive.
- Return fewer roles rather than weakening a hard gate to fill a quota.
- Apply level, language, frontend/framework dependence, domain, authorization,
  location, and evidence gates before scoring.
- Use official ATS or employer pages for final inclusion. Record exact canonical
  URLs and verification timestamps.
- Use search engines, aggregators, and job boards for discovery only.
- Deduplicate by official ATS identifier, then canonical official URL, then
  company/title/location similarity warnings.
- Label evidence as `resume`, `verified_artifact`, or `user_confirmed`.
- Keep explicit skip reasons and retain enough identity to prevent rediscovery.
- Never bypass access controls or claim an ambiguous page is live.

Read `standards/evaluation.md` before classifying or ranking a result. Class is
authoritative; numerical score ranks only within a class.

## Reports, feedback, and application state

Read `standards/report.md` before generating or changing a report. Job Deck
records explicit feedback; the server does not infer career preferences. Treat
new feedback as pending agent work.

Before ending a relevant session, or at the start of the next one:

1. Review pending feedback and the associated official posting.
2. Identify plausible preference implications.
3. Ask the user when the implication is materially ambiguous.
4. Update `private/search-preferences.md` with confirmed lessons and its last
   update date.
5. Mark the feedback reviewed and record the outcome in the activity log.

Opening a link means `Opened`, never `Applied`. Record `Applied` only after an
explicit user action or direct statement. Do not submit applications, complete
assessments, contact people, or answer demographic, legal, salary,
authorization, or sponsorship questions on the user's behalf.

## Logging contract

`private/logs/events.jsonl` is the append-only machine record.
`private/logs/log.md` is the compact human-readable projection. Log meaningful
workflow events, not every browser action or sensitive resume content.

- `👤 User` — explicit user actions or corrections.
- `🤖 Agent` — searches, evaluation runs, preference reviews, and report creation.
- `⚙️ System` — runtime builds, imports, failures, and integrity checks.

Use `---` between entries. A search entry should summarize the request, research
mode, candidates reviewed, roles passing gates, roles reported, source coverage,
and important skip categories. A user application entry must say
`self-reported` unless Job Deck received the explicit action directly. Never
infer an application from an opened page or email search.

Keep `log.md` scannable by consolidating repetitive entries into dated
summaries. Update `Last consolidated` at the top when consolidation occurs.
Never delete the corresponding append-only JSONL events.

## Selective source maintenance

Read `src/README.md` before changing source. Do not build source for operational
job-search changes.

| Change | Required action |
|---|---|
| Resume, preferences, reports, feedback, logs, standards, or examples | No build |
| Go server source | Run Go tests and rebuild Go only |
| Vue source, CSS, or web public assets | Run Bun checks/build, then rebuild Go |
| Committed `src/web/dist/` already matches the web source | Never rebuild web during setup |

Stop only Job Deck's recorded process, test the affected layer, replace the
executable atomically, verify source hashes, and restart only if it was
previously running.
