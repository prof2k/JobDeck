---
last_updated: 2026-08-05
status: living
---

# Report standard

Job Deck is the durable output surface for the workflow. When the local runtime
is available, one dashboard contains immutable search-batch records plus mutable
progress and feedback. When it is unavailable, each batch becomes one
self-contained HTML report. The agent performs the work; the report surface is
where the user reviews and acts on the result.

This standard evolves when a reusable report or interaction rule improves.
Candidate-specific display preferences remain under `private/`.

## Report library

Show report title, calibration or full-search type, creation time, last-opened
time, job count, Ready count, and application progress. The newest report leads.

The library belongs only to the local dashboard. A standalone report opens
directly to its one batch and must not imitate a multi-report database.

## Report detail

Use a ranked whole-row-clickable ledger with a persistent evidence rail. The
selected job must show:

- company, title, location, arrangement, salary, and official source;
- fit class, score, status, and verification timestamp;
- concise recommendation;
- requirements and direct, adjacent, or missing evidence;
- gaps, risks, and relevant verified company-application policy;
- Open official application, explicit Applied, and Skip controls;
- calibration feedback controls when applicable.

Opening the official page changes the state to `Opened`, never `Applied`.

For the first feedback batch, place **Your feedback** directly inside each
selected job's detail, immediately after the recommendation. Ask “Is this the
kind of job you want?” and provide Strong fit, Maybe, Wrong fit, and Never show
similar choices plus optional context. Make the section visually distinct with
a slightly lighter surface rather than a stronger border.

These controls appear only for calibration reports in the full local dashboard.
Standard search reports never show them. They are optional and self-discoverable:
do not direct the user to operate them or require all jobs to be rated before
another search. The primary calibration prompt is conversational and broad:
ask which roles stand out as good, wrong, or uninteresting and accept whatever
level of detail the user naturally provides.

The report page begins with a compact vertical trail: Reports, report type,
title, and summary. Reports must sit above the batch label rather than beside it.
On desktop, the workspace should occupy roughly 90% of the viewport width and
split into a compact status-rich job ledger on the left and a larger selected-job
surface on the right. Give both panes the same working height with independent
scrolling, and keep the selected job's application actions pinned. Below the
desktop breakpoint, stack the ledger above the selected-job surface.

## Offline behavior

Cache the application shell and last successfully loaded report snapshots. When
the service is unavailable, show the cached data read-only with a clear offline
banner. Do not imply that links, verification times, or progress are current.

## Standalone report behavior

Use a standalone HTML report whenever the current environment cannot run and
open the local dashboard, regardless of which runtime or host capability is
missing. Treat this as a normal delivery mode.

- Produce one self-contained `.html` file per completed batch.
- Embed the private structured report data safely inside the HTML. Never expose,
  attach, or require the user to open a JSON file.
- Do not fetch a sibling data file and do not require localhost, a package
  install, external assets, a framework runtime, SQLite, or IndexedDB.
- Preserve the ranked ledger and selected-job evidence layout. Stack it on
  smaller screens.
- Do not show marking, reaction, or feedback controls in any standalone report,
  including calibration. Ask for calibration feedback in the conversation.
- Optimize calibration and normal standalone reports for reading, comparison,
  and opening official application links.
- Show existing application states when supplied, but do not imply that a
  standalone page is the authoritative application ledger.
- Keep all candidate-bearing HTML under `private/reports/`. Present the HTML
  result to the user without narrating its internal data or runtime choice.

An optional top-level application helper controls the one-time autofill
recommendation:

```json
{
  "applicationHelper": {
    "show": true,
    "title": "Make applications easier",
    "message": "Install Simplify Copilot to make filling out job applications easier.",
    "linkLabel": "Install Simplify Copilot",
    "url": "https://chromewebstore.google.com/detail/simplify-copilot-autofill/pbanhockgagggenencehbnadejlgchfc"
  }
}
```

When `show` is true, the report may display the quiet helper callout and the
agent should mention it once when delivering the report. Absence or `false`
means no recommendation. The extension is optional and installation remains the
user's action.

## Accessibility

Rows must be keyboard accessible. Maintain visible focus, semantic headings,
reduced-motion support, meaningful empty and error states, and readable contrast.
