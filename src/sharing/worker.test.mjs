import { expect, test } from 'bun:test';
import worker, { sanitizeSnapshot } from './worker.mjs';

function fixture() {
  let stored = null;
  let revision = 0;
  const env = {
    VIEWER_PASSWORD: 'fictional-viewer-password',
    SESSION_SECRET: 'fictional-session-secret-1234567890',
    PUBLISH_TOKEN: 'fictional-publish-token-1234567890',
    LOGIN_LIMITER: { limit: async () => ({ success: true }) },
    ASSETS: { fetch: async () => new Response('<html>Viewer shell</html>') },
    PROGRESS: {
      async get() {
        if (stored === null) return null;
        return { etag: String(revision), body: stored, json: async () => JSON.parse(stored) };
      },
      async put(name, value, options) {
        if (options.onlyIf.etagMatches && options.onlyIf.etagMatches !== String(revision)) return null;
        stored = value;
        revision += 1;
        return { etag: String(revision) };
      },
    },
  };
  const send = (path, init = {}) => worker.fetch(new Request(`https://progress.example${path}`, init), env);
  const login = () => send('/api/login', { method: 'POST', headers: { Origin: 'https://progress.example', 'Content-Type': 'application/json' }, body: JSON.stringify({ password: env.VIEWER_PASSWORD }) });
  const publish = (body) => send('/api/progress', { method: 'PUT', headers: { Authorization: `Bearer ${env.PUBLISH_TOKEN}`, 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  return { env, send, login, publish };
}

function snapshot() {
  return { version: 1, timezone: 'UTC', today: '2026-10-28', updatedAt: '2026-10-28T12:00:00Z', tasks: [], completions: [], targets: [], days: [{ date: '2026-10-28', applications: 12 }] };
}

test('anonymous readers cannot access progress and a viewer cannot publish', async () => {
  const { send, login } = fixture();
  expect((await send('/api/progress')).status).toBe(401);
  const response = await login();
  expect(response.status).toBe(200);
  const cookie = response.headers.get('Set-Cookie');
  expect(cookie).toContain('HttpOnly; Secure; SameSite=Strict');
  const attempt = await send('/api/progress', { method: 'PUT', headers: { Cookie: cookie.split(';')[0], 'Content-Type': 'application/json' }, body: JSON.stringify(snapshot()) });
  expect(attempt.status).toBe(403);
});

test('publisher upload and authenticated viewer read preserve only approved fields', async () => {
  const { publish, login, send } = fixture();
  expect((await publish({ ...snapshot(), privateNotes: 'must never be shared' })).status).toBe(200);
  const session = await login();
  const cookie = session.headers.get('Set-Cookie').split(';')[0];
  const response = await send('/api/progress', { headers: { Cookie: cookie } });
  expect(response.status).toBe(200);
  expect(response.headers.get('Cache-Control')).toBe('no-store');
  const data = await response.json();
  expect(data.days[0].applications).toBe(12);
  expect(data.privateNotes).toBeUndefined();
  const logout = await send('/api/logout', { method: 'POST', headers: { Origin: 'https://progress.example', Cookie: cookie } });
  expect(logout.headers.get('Set-Cookie')).toContain('Max-Age=0');
});

test('incorrect passwords, forged sessions, and cross-origin login fail', async () => {
  const { send } = fixture();
  const incorrect = await send('/api/login', { method: 'POST', headers: { Origin: 'https://progress.example', 'Content-Type': 'application/json' }, body: '{"password":"wrong"}' });
  expect(incorrect.status).toBe(401);
  expect((await send('/api/progress', { headers: { Cookie: `__Host-jobdeck_viewer=9999999999.${'0'.repeat(64)}` } })).status).toBe(401);
  expect((await send('/api/login', { method: 'POST', headers: { Origin: 'https://unrelated.example' } })).status).toBe(403);
});

test('missing configuration and login rate limiting fail closed', async () => {
  const { env, send, login } = fixture();
  env.LOGIN_LIMITER.limit = async () => ({ success: false });
  expect((await login()).status).toBe(429);
  env.SESSION_SECRET = '';
  expect((await send('/api/progress')).status).toBe(503);
});

test('malformed dates, oversize uploads, and old snapshots are rejected', async () => {
  const { publish, send, env } = fixture();
  expect((await publish({ ...snapshot(), today: '2026-02-30' })).status).toBe(400);
  expect((await publish({ ...snapshot(), tasks: [null] })).status).toBe(400);
  const oversized = await send('/api/progress', { method: 'PUT', headers: { Authorization: `Bearer ${env.PUBLISH_TOKEN}`, 'Content-Type': 'application/json' }, body: 'x'.repeat(1024 * 1024 + 1) });
  expect(oversized.status).toBe(400);
  expect((await publish(snapshot())).status).toBe(200);
  expect((await publish({ ...snapshot(), updatedAt: '2026-10-27T12:00:00Z' })).status).toBe(409);
  expect(sanitizeSnapshot(snapshot())).toEqual(snapshot());
});
