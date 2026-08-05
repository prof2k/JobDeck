PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS reports (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL CHECK (kind IN ('calibration', 'search')),
  title TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  imported_at TEXT NOT NULL,
  last_opened_at TEXT
);

CREATE TABLE IF NOT EXISTS jobs (
  id INTEGER PRIMARY KEY,
  canonical_key TEXT NOT NULL UNIQUE,
  canonical_url TEXT NOT NULL,
  company TEXT NOT NULL,
  title TEXT NOT NULL,
  location TEXT NOT NULL DEFAULT 'Not listed',
  arrangement TEXT NOT NULL DEFAULT 'Not listed',
  salary TEXT NOT NULL DEFAULT 'Not listed',
  source TEXT NOT NULL,
  live_verified_at TEXT NOT NULL,
  current_state TEXT NOT NULL DEFAULT 'ready'
    CHECK (current_state IN ('ready', 'opened', 'applied', 'skipped', 'expired', 'rejected')),
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS report_jobs (
  report_id TEXT NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
  job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE RESTRICT,
  rank INTEGER NOT NULL CHECK (rank > 0),
  fit_class TEXT NOT NULL CHECK (fit_class IN ('A', 'B', 'C', 'D')),
  score INTEGER CHECK (score IS NULL OR (score >= 0 AND score <= 100)),
  recommendation TEXT NOT NULL,
  evaluation_json TEXT NOT NULL,
  PRIMARY KEY (report_id, job_id),
  UNIQUE (report_id, rank)
);

CREATE TABLE IF NOT EXISTS job_events (
  id INTEGER PRIMARY KEY,
  job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  event_type TEXT NOT NULL
    CHECK (event_type IN ('ready', 'opened', 'applied', 'skipped', 'expired', 'rejected')),
  occurred_at TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS job_feedback (
  id INTEGER PRIMARY KEY,
  job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  report_id TEXT REFERENCES reports(id) ON DELETE SET NULL,
  verdict TEXT NOT NULL CHECK (verdict IN ('strong_fit', 'maybe', 'wrong_fit', 'never_show')),
  reason TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  reviewed_at TEXT,
  review_note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS company_policies (
  id INTEGER PRIMARY KEY,
  company_key TEXT NOT NULL,
  summary TEXT NOT NULL,
  maximum_applications INTEGER,
  window_days INTEGER,
  source_url TEXT NOT NULL,
  verified_at TEXT NOT NULL,
  UNIQUE (company_key, source_url)
);

CREATE INDEX IF NOT EXISTS idx_report_jobs_report_rank
  ON report_jobs(report_id, rank);
CREATE INDEX IF NOT EXISTS idx_jobs_company_title
  ON jobs(company, title);
CREATE INDEX IF NOT EXISTS idx_job_events_job_time
  ON job_events(job_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_feedback_job_time
  ON job_feedback(job_id, created_at DESC);
