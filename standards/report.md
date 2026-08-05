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

## Offline behavior

Cache the application shell and last successfully loaded report snapshots. When
the service is unavailable, show the cached data read-only with a clear offline
banner. Do not imply that links, verification times, or progress are current.

## Accessibility

Rows must be keyboard accessible. Maintain visible focus, semantic headings,
reduced-motion support, meaningful empty and error states, and readable contrast.
