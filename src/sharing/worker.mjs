const encoder = new TextEncoder();
const cookieName = '__Host-jobdeck_viewer';
const sessionSeconds = 7 * 24 * 60 * 60;
const maxBytes = 1024 * 1024;

function json(body, status = 200, headers = {}) {
  return Response.json(body, { status, headers });
}

async function key(secret) {
  return crypto.subtle.importKey('raw', encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' }, false, ['sign', 'verify']);
}

async function signature(value, secret) {
  const bytes = new Uint8Array(await crypto.subtle.sign('HMAC', await key(secret), encoder.encode(value)));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

async function matches(value, expected, secret) {
  const hex = await signature(expected, secret);
  const bytes = Uint8Array.from(hex.match(/../g), (pair) => Number.parseInt(pair, 16));
  return crypto.subtle.verify('HMAC', await key(secret), bytes, encoder.encode(value));
}

async function validSession(request, secret) {
  const token = request.headers.get('Cookie')?.split(';').map((part) => part.trim())
    .find((part) => part.startsWith(`${cookieName}=`))?.slice(cookieName.length + 1) ?? '';
  const [expires, hex] = token.split('.');
  const expiry = Number(expires);
  const now = Math.floor(Date.now() / 1000);
  if (!Number.isInteger(expiry) || expiry <= now || expiry > now + sessionSeconds
    || !/^[a-f0-9]{64}$/.test(hex ?? '')) return false;
  const bytes = Uint8Array.from(hex.match(/../g), (pair) => Number.parseInt(pair, 16));
  return crypto.subtle.verify('HMAC', await key(secret), bytes, encoder.encode(expires));
}

async function readJSON(request) {
  if (!request.headers.get('Content-Type')?.startsWith('application/json')) {
    throw new Error('Send application/json.');
  }
  const reader = request.body?.getReader();
  if (!reader) throw new Error('A JSON body is required.');
  const chunks = [];
  let size = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    size += value.byteLength;
    if (size > maxBytes) { await reader.cancel(); throw new Error('Snapshot exceeds 1 MB.'); }
    chunks.push(value);
  }
  const bytes = new Uint8Array(size);
  let offset = 0;
  chunks.forEach((chunk) => { bytes.set(chunk, offset); offset += chunk.length; });
  return JSON.parse(new TextDecoder().decode(bytes));
}

function text(value, limit = 200) {
  if (typeof value !== 'string' || value.length > limit) throw new Error('Invalid snapshot text.');
  return value;
}

function date(value) {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)
    || !Number.isFinite(Date.parse(`${value}T00:00:00Z`))
    || new Date(`${value}T00:00:00Z`).toISOString().slice(0, 10) !== value) {
    throw new Error('Invalid snapshot date.');
  }
  return value;
}

function integer(value, max = 100000) {
  if (!Number.isInteger(value) || value < 0 || value > max) throw new Error('Invalid snapshot count.');
  return value;
}

function list(value, max = 20000) {
  if (!Array.isArray(value) || value.length > max
    || value.some((entry) => !entry || typeof entry !== 'object' || Array.isArray(entry))) {
    throw new Error('Invalid snapshot list.');
  }
  return value;
}

function officialURL(value) {
  const candidate = text(value, 4096);
  let parsed;
  try { parsed = new URL(candidate); } catch { throw new Error('Invalid report URL.'); }
  if (!['https:', 'http:'].includes(parsed.protocol)) throw new Error('Invalid report URL.');
  return candidate;
}

function textList(value) {
  if (!Array.isArray(value) || value.length > 200) throw new Error('Invalid report list.');
  return value.map((item) => text(item, 10000));
}

function sanitizeEvaluation(input) {
  if (!input || typeof input !== 'object') throw new Error('Invalid report evidence.');
  const result = {
    requirements: list(input.requirements, 500).map((item) => ({
      requirement: text(item.requirement, 4000), match: text(item.match, 64),
      evidence: text(item.evidence, 10000), source: text(item.source, 2000),
    })),
    gaps: textList(input.gaps), risks: textList(input.risks),
  };
  if (input.roleType) result.roleType = text(input.roleType);
  if (['pass', 'maybe'].includes(input.screeningVerdict)) result.screeningVerdict = input.screeningVerdict;
  if (input.companyPolicy) {
    const policy = input.companyPolicy;
    result.companyPolicy = {
      summary: text(policy.summary, 10000), sourceUrl: officialURL(policy.sourceUrl),
      verifiedAt: text(policy.verifiedAt, 40),
      ...(policy.maximumApplications === undefined ? {} : { maximumApplications: integer(policy.maximumApplications) }),
      ...(policy.windowDays === undefined ? {} : { windowDays: integer(policy.windowDays) }),
    };
  }
  return result;
}

function sanitizeReport(input) {
  if (input === null) return null;
  if (!input?.report || !['search', 'calibration'].includes(input.report.kind)) throw new Error('Invalid report.');
  const jobs = list(input.jobs, 1000).map((job) => {
    if (!['ready', 'opened', 'applied', 'skipped', 'expired', 'rejected'].includes(job.state)
      || !['A', 'B', 'C', 'D'].includes(job.fitClass)) throw new Error('Invalid report job.');
    return {
      id: integer(job.id, Number.MAX_SAFE_INTEGER), canonicalKey: text(job.canonicalKey, 512),
      company: text(job.company, 500), title: text(job.title, 1000), location: text(job.location, 1000),
      arrangement: text(job.arrangement, 500), salary: text(job.salary, 1000),
      applicationUrl: officialURL(job.applicationUrl), source: text(job.source, 2000),
      liveVerifiedAt: text(job.liveVerifiedAt, 40), state: job.state, rank: integer(job.rank, 1000),
      fitClass: job.fitClass, score: job.score === null ? null : integer(job.score, 100),
      recommendation: text(job.recommendation, 20000), evaluation: sanitizeEvaluation(job.evaluation),
      feedbackVerdict: null, feedbackReason: null, feedbackNote: null,
    };
  });
  const report = input.report;
  if (integer(report.jobCount, 1000) !== jobs.length
    || new Set(jobs.map((job) => job.canonicalKey)).size !== jobs.length
    || new Set(jobs.map((job) => job.rank)).size !== jobs.length) throw new Error('Invalid report membership.');
  return {
    report: {
      id: text(report.id, 512), kind: report.kind, title: text(report.title, 1000),
      summary: text(report.summary, 20000), createdAt: text(report.createdAt, 40), lastOpenedAt: null,
      groupBy: report.groupBy === 'roleType' ? 'roleType' : 'company', jobCount: jobs.length,
      readyCount: jobs.filter((job) => ['ready', 'opened'].includes(job.state)).length,
      appliedCount: jobs.filter((job) => job.state === 'applied').length,
    },
    jobs,
  };
}

// The owner explicitly shares the latest batch. Private feedback and unknown fields are stripped.
export function sanitizeSnapshot(input) {
  if (!input || input.version !== 1 || input.timezone !== 'UTC'
    || typeof input.updatedAt !== 'string' || !Number.isFinite(Date.parse(input.updatedAt))) {
    throw new Error('Unsupported progress snapshot.');
  }
  return {
    version: 1, timezone: input.timezone, updatedAt: text(input.updatedAt, 40), today: date(input.today),
    tasks: list(input.tasks, 1000).map((task) => {
      if (!['once', 'daily', 'weekly', 'monthly', 'quarterly'].includes(task.recurrence)) throw new Error('Invalid recurrence.');
      return { id: text(task.id, 64), title: text(task.title), recurrence: task.recurrence,
        startsOn: date(task.startsOn), archivedOn: task.archivedOn === null ? null : date(task.archivedOn) };
    }),
    completions: list(input.completions).map((entry) => ({
      taskId: text(entry.taskId, 64), period: text(entry.period, 20), completedOn: date(entry.completedOn),
    })),
    targets: list(input.targets, 10000).map((entry) => ({ effectiveOn: date(entry.effectiveOn), target: integer(entry.target, 1000) })),
    days: list(input.days, 10000).map((entry) => ({ date: date(entry.date), applications: integer(entry.applications) })),
    ...(input.latestReport === undefined ? {} : { latestReport: sanitizeReport(input.latestReport) }),
  };
}

async function route(request, env) {
  const { pathname, origin } = new URL(request.url);
  if (!pathname.startsWith('/api/')) return env.ASSETS.fetch(request);
  if (pathname === '/api/health') return json({ app: 'jobdeck', mode: 'shared' });
  if (!env.PROGRESS || env.VIEWER_PASSWORD?.length < 12 || !env.VIEWER_PASSWORD
    || env.SESSION_SECRET?.length < 32 || !env.SESSION_SECRET
    || env.VIEWER_PASSWORD.length > 256
    || env.PUBLISH_TOKEN?.length < 32 || !env.PUBLISH_TOKEN) {
    return json({ message: 'Sharing is not configured yet.' }, 503);
  }
  const requestOrigin = request.headers.get('Origin');
  if (requestOrigin && requestOrigin !== origin) return json({ message: 'Unexpected origin.' }, 403);
  if (pathname === '/api/login' && request.method === 'POST') {
    if (requestOrigin !== origin) return json({ message: 'Unexpected origin.' }, 403);
    if (!env.LOGIN_LIMITER) return json({ message: 'Login protection is not configured.' }, 503);
    const { success } = await env.LOGIN_LIMITER.limit({ key: request.headers.get('CF-Connecting-IP') ?? 'unknown' });
    if (!success) return json({ message: 'Too many attempts. Try again in a minute.' }, 429, { 'Retry-After': '60' });
    const body = await readJSON(request);
    if (typeof body?.password !== 'string' || body.password.length > 256
      || !await matches(body.password, env.VIEWER_PASSWORD, env.SESSION_SECRET)) {
      return json({ message: 'Incorrect password.' }, 401);
    }
    const expires = String(Math.floor(Date.now() / 1000) + sessionSeconds);
    const token = `${expires}.${await signature(expires, env.SESSION_SECRET)}`;
    return json({ ok: true }, 200, { 'Set-Cookie': `${cookieName}=${token}; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=${sessionSeconds}` });
  }
  if (pathname === '/api/logout' && request.method === 'POST') {
    if (requestOrigin !== origin) return json({ message: 'Unexpected origin.' }, 403);
    return json({ ok: true }, 200, { 'Set-Cookie': `${cookieName}=; Path=/; HttpOnly; Secure; SameSite=Strict; Max-Age=0` });
  }
  if (pathname === '/api/progress' && request.method === 'GET') {
    if (!await validSession(request, env.SESSION_SECRET)) return json({ message: 'Enter the viewer password.' }, 401);
    const object = await env.PROGRESS.get('progress.json');
    if (!object) return json({ message: 'No progress has been published yet. Check back after the first update.' }, 404);
    return new Response(object.body, { headers: { 'Content-Type': 'application/json; charset=utf-8' } });
  }
  if (pathname === '/api/progress' && request.method === 'PUT') {
    const bearer = request.headers.get('Authorization') ?? '';
    if (!await matches(bearer, `Bearer ${env.PUBLISH_TOKEN}`, env.SESSION_SECRET)) {
      return json({ message: 'Invalid publishing credential.' }, 403);
    }
    const snapshot = sanitizeSnapshot(await readJSON(request));
    const previous = await env.PROGRESS.get('progress.json');
    if (previous && Date.parse((await previous.json()).updatedAt) > Date.parse(snapshot.updatedAt)) {
      return json({ message: 'A newer snapshot is already published.' }, 409);
    }
    const saved = await env.PROGRESS.put('progress.json', JSON.stringify(snapshot), {
      httpMetadata: { contentType: 'application/json' },
      onlyIf: previous ? { etagMatches: previous.etag } : { etagDoesNotMatch: '*' },
    });
    if (!saved) return json({ message: 'Progress changed during publication. Retry publishing.' }, 409);
    return json({ ok: true, updatedAt: snapshot.updatedAt });
  }
  return json({ message: 'Not found.' }, 404);
}

export default {
  async fetch(request, env) {
    let response;
    try { response = await route(request, env); }
    catch (error) {
      if (error instanceof SyntaxError || error?.message?.startsWith('Invalid')
        || error?.message?.startsWith('Unsupported') || error?.message?.startsWith('Send ')
        || error?.message === 'Snapshot exceeds 1 MB.' || error?.message === 'A JSON body is required.') {
        response = json({ message: error.message }, 400);
      } else { response = json({ message: 'Could not load or save progress. Try again.' }, 503); }
    }
    const secured = new Response(response.body, response);
    secured.headers.set('Cache-Control', 'no-store');
    secured.headers.set('X-Content-Type-Options', 'nosniff');
    secured.headers.set('Referrer-Policy', 'no-referrer');
    secured.headers.set('X-Frame-Options', 'DENY');
    secured.headers.set('Content-Security-Policy', "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'");
    return secured;
  },
};
