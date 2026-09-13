CREATE TABLE IF NOT EXISTS repositories (
    id TEXT PRIMARY KEY,
    ref TEXT NOT NULL,
    demo INTEGER NOT NULL CHECK (demo IN (0, 1)),
    UNIQUE(ref, demo)
);
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL REFERENCES repositories(id),
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    error TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS one_active_job ON jobs(repository_id) WHERE status IN ('queued', 'running');
CREATE INDEX IF NOT EXISTS jobs_pending ON jobs(status, created_at);
CREATE INDEX IF NOT EXISTS jobs_repository ON jobs(repository_id, created_at DESC);
CREATE TABLE IF NOT EXISTS reports (
    repository_id TEXT PRIMARY KEY REFERENCES repositories(id),
    report_json TEXT NOT NULL,
    snapshot_json TEXT NOT NULL
);
PRAGMA user_version = 1;
