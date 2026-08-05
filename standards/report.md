---
last_updated: 2026-08-05
status: living
---

# Report standard

Job Deck is the durable output surface for the workflow: one local application
containing many immutable search-batch records plus mutable progress and
feedback. The agent performs the work, but the dashboard is where the user
reviews and acts on its results.

This standard evolves when a reusable report or interaction rule improves.
Candidate-specific display preferences remain under `private/`.

## Report library

Show report title, calibration or full-search type, creation time, last-opened
time, job count, Ready count, and application progress. The newest report leads.

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

Feedback is optional. Tell the user where the controls are, invite feedback on
the jobs that produce a clear reaction, and allow every role to be skipped
without friction. Never require all jobs to be rated before another search.

The report page begins with a compact vertical trail: Reports, report type,
title, and summary. Reports must sit above the batch label rather than beside it.
The selected-job surface should occupy roughly 90% of the viewport width on
desktop, remain internally scrollable, and keep application actions pinned.

## Offline behavior

Cache the application shell and last successfully loaded report snapshots. When
the service is unavailable, show the cached data read-only with a clear offline
banner. Do not imply that links, verification times, or progress are current.

## Accessibility

Rows must be keyboard accessible. Maintain visible focus, semantic headings,
reduced-motion support, meaningful empty and error states, and readable contrast.
