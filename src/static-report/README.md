---
last_updated: 2026-08-05
status: living
---

# Standalone report

This folder contains the dependency-free report surface used when the current
host cannot run or open Job Deck's local dashboard. It is a normal delivery
mode, not an error page.

`template.html` is one complete HTML document with inline CSS and JavaScript. It
contains exactly one marker:

```text
__JOB_DECK_REPORT_DATA__
```

Replace that marker with JSON-encoded report data after escaping `<`, `>`, `&`,
U+2028, and U+2029 as Unicode escapes. Write the finished candidate-bearing file
under `private/reports/` and present only that HTML file to the user. Never
attach the structured source data or make the page fetch a sibling file.

`../scripts/render-static-report.mjs` performs this replacement when an existing
Node.js or Bun runtime happens to be available. It is a maintainer convenience,
not a runtime requirement. An agent with file-writing capability can perform the
same single-marker replacement directly.

The template chooses its interface from `report.kind`:

- `calibration` labels the batch as feedback-oriented but remains a read-only
  evidence report; the agent asks for the user's reaction in conversation;
- every other kind is a consumption-first search report using the same reading
  surface.

Both modes are single-report pages with a ranked ledger and one selected-job
evidence sheet. They require no localhost, database, IndexedDB, package install,
external font, framework runtime, or marking and feedback controls.

When the report's optional top-level `applicationHelper.show` value is `true`,
the page also renders its configured Simplify Copilot recommendation and official
installation link. The template does not hard-code or independently trigger the
recommendation.
