# Shared progress

An opt-in, read-only progress viewer. The Vue static build and Worker deploy
together; a private R2 bucket holds one `progress.json` snapshot. The local
JobDeck ledger remains authoritative. The published schema contains task titles,
recurrence, completion dates, target history, and dated application counts. It
does not contain reports, job descriptions, resumes, or research notes.

## Local verification

Install the existing web dependencies with `bun install --frozen-lockfile` from
`src/web`. From `src/sharing`, run `bun install --frozen-lockfile`, then
`bun run test` and `bun run build`. The web source has additional tests in `../web/tests`. The shared static output is versioned, matching the local dashboard convention, so the exact tested build can be deployed from the source revision.
Run the Go tests from `../server` with `go test -race ./...`.

For the Cloudflare local runtime, create `private/runtime/sharing-dev.env` at
the project root, with file mode 0600 and these three independently generated
values. Keep this file and every other credential under ignored `private/`:

- `VIEWER_PASSWORD`: at least 12 characters, at most 256.
- `SESSION_SECRET`: at least 32 characters; signs expiring viewer cookies.
- `PUBLISH_TOKEN`: at least 32 characters; used only by the local publisher.

Then run, from `src/sharing`:

```sh
bun run dev --local --ip 127.0.0.1 --port 8787 \
  --persist-to ../../private/runtime/cloudflare \
  --env-file ../../private/runtime/sharing-dev.env
```

Local R2 persistence stays under `private/runtime/`. Local tests must never use
`--remote`. Sessions use Secure, HttpOnly cookies; use localhost or local HTTPS
for browser login testing. The unit tests use synthetic data and an R2 stand-in;
verify the real local Worker/R2 integration separately before deployment.

## Live setup (requires explicit deployment authorization)

1. Activate Workers & Pages in the chosen Cloudflare account, including its
   `workers.dev` subdomain. Confirm there is no existing Worker or bucket with
   the configured names before creating them.
2. Create a private R2 bucket named `jobdeck-progress` (or update the binding in
   `wrangler.jsonc`). Leave its public URL and custom-domain access disabled.
3. Build with `bun run build`. Configure the three production secrets using
   `wrangler secret put VIEWER_PASSWORD`, `wrangler secret put SESSION_SECRET`,
   and `wrangler secret put PUBLISH_TOKEN` in this directory. Production secrets
   must be independent of the local development credentials.
4. Deploy with `bun run deploy`. The configuration invokes the Worker for every
   `/api/*` request, including browser navigation. The viewer bundle uses a
   separate entry point and does not register the local offline service worker.
5. Store the deployed site's base URL and production publishing token in the
   ignored `private/sharing.json` configuration (or set them through the local
   authenticated configuration API). This is deployment setup, not a product
   action. The editor has no sharing or publishing controls.
6. Check anonymous data requests fail, viewer login works, viewers cannot
   publish, the first snapshot appears, and a local edit updates the shared page
   after refresh. Verify there are no unexpected public bucket access paths.

The login rate limiter is approximate and applies per Cloudflare location.
Viewer sessions expire after seven days; rotating `SESSION_SECRET` invalidates
existing cookies. The publisher validates JSON and strips unknown fields before
writing R2. Conditional writes reject overlapping or stale uploads. Responses
use `Cache-Control: no-store`; authenticated snapshots are not cached offline.

## Progress semantics

- Dates and application-day boundaries use UTC. The viewer displays update timestamps in UTC.
- Applications count distinct jobs by their first explicit Applied event.
  Opening a job does not count. Undated legacy application states are not
  assigned invented dates.
- Daily tasks reset each day; weekly tasks use Monday-start weeks; monthly and
  quarterly tasks use calendar periods. One-time tasks keep their completion
  until archived. Archival preserves earlier history.
- The daily application target autosaves in Settings and takes effect from today. Existing earlier targets
  and completions are retained. A zero target means no target was set.
- The seven-day application average includes only days with a positive target,
  including zero-application days, with each day's contribution capped at 100%.
- Changes save locally first and notify a single background publisher immediately.
  The UI does not wait for the upload. Failed uploads retry automatically. A
  two-second local reconciliation check also catches direct imports, updates from
  another local dashboard, and UTC rollover. Unchanged snapshots do not generate
  cloud writes. Successful snapshot hashes persist across restarts.
- The viewer reads the last published snapshot on opening, returning to the
  tab, or pressing Refresh. It remains available when the local computer is off.

Cloudflare references: [Static Assets](https://developers.cloudflare.com/workers/static-assets/),
[R2 bindings](https://developers.cloudflare.com/r2/api/workers/workers-api-reference/),
[R2 consistency](https://developers.cloudflare.com/r2/reference/consistency/), and
[login rate limiting](https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/).
