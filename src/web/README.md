---
last_updated: 2026-08-05
status: living
---

# Job Deck web interface

This folder contains the Vue interface source and the committed production
output consumed by the Go server. Read this file only when changing the
interface; operational job searches do not require it.

Do not edit `dist/` directly. Change the Vue source or public assets, run the
type check and production build described in `../README.md`, then rebuild the Go
executable. The committed output lets a normal Job Deck user avoid installing
Bun or Node.js.

## Design contract: quiet evidence desk

The person opens Job Deck after a job-search agent has researched a batch. They
need to decide which jobs deserve an application, understand why each one fits,
and preserve progress without reopening a spreadsheet or re-reading the
posting.

The interface should feel like a calm late-night research desk: dense enough to
support serious comparison and quiet enough that evidence leads.

### Domain and signature

The domain is search batches, shortlists, evidence, constraints, freshness,
application history, progress stamps, and calibration corrections.

The wide evidence sheet is the signature. Ranked roles remain a compact ledger;
the selected role expands below it into requirement-level evidence, gaps,
application state, and an unmistakable in-context feedback surface. The same
sheet remains legible in an offline snapshot.

### Visual system

- Charcoal desk: application canvas.
- Graphite: inset controls and secondary surfaces.
- Soft paper: primary text.
- Violet highlighter: selection and primary action.
- Amber note: uncertainty, adjacency, and stale data.
- Green stamp: explicit completion.
- Depth: quiet surface shifts lead; reserve borders for controls and necessary
  separators, with no decorative shadows.
- Surfaces: desk `#0f0f11`, sheet `#19191d`, raised `#202026`, inset `#121216`.
- Text: paper `#f0eff4`, secondary `#c2c0c8`, tertiary `#96949e`, muted
  `#6f6d77`.
- Accent: violet `#a9adff`; caution `#e6c86e`; success `#75c89a`; danger
  `#e98b9f`.
- Typography: locally bundled DM Sans for prose, labels, and headings; system
  monospace only for ranks, hashes, and timestamps.
- Spacing: 4px base unit, primarily 8, 12, 16, 24, and 32px.
- Radius: 6px controls and 10px panels; no pills except compact status badges.
- Motion: 140–180ms ease-out, removed when requested by the operating system.

### Rejected defaults

- Replace a generic sidebar plus detached dashboard cards with a report shelf
  and a wide, stacked evidence-first workspace.
- Avoid rainbow fit scores; use one violet accent plus semantic amber and green.
- Avoid card-grid job browsing; use ranked rows so comparison remains fast.
- Avoid a detached calibration form; ask for optional feedback inside the
  selected job's detail, directly below the recommendation.
