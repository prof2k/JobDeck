import { rm, writeFile } from 'node:fs/promises';

// The read-only viewer never installs the local app's offline service worker.
await rm(new URL('./dist/sw.js', import.meta.url), { force: true });
await writeFile(new URL('./dist/_headers', import.meta.url), `/*
  X-Frame-Options: DENY
  X-Content-Type-Options: nosniff
  Referrer-Policy: no-referrer
  X-Robots-Tag: noindex, nofollow
  Content-Security-Policy: default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'
`);
