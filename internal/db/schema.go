package db

const schema = `
CREATE TABLE IF NOT EXISTS repos (
	id           TEXT PRIMARY KEY,
	name         TEXT NOT NULL,
	url          TEXT NOT NULL UNIQUE,
	enabled      INTEGER NOT NULL DEFAULT 1,
	priority     INTEGER NOT NULL DEFAULT 0,
	last_fetched TEXT,
	etag         TEXT,
	created_at   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS plugins (
	id          TEXT PRIMARY KEY,
	repo_id     TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
	guid        TEXT NOT NULL,
	name        TEXT NOT NULL,
	description TEXT,
	overview    TEXT,
	owner       TEXT,
	category    TEXT,
	UNIQUE(repo_id, guid)
);

CREATE TABLE IF NOT EXISTS plugin_versions (
	id             TEXT PRIMARY KEY,
	plugin_id      TEXT NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
	version        TEXT NOT NULL,
	changelog      TEXT,
	target_abi     TEXT,
	source_url     TEXT NOT NULL,
	checksum       TEXT NOT NULL,
	timestamp      TEXT,
	local_path     TEXT,
	download_status TEXT NOT NULL DEFAULT 'pending',
	downloaded_at  TEXT,
	UNIQUE(plugin_id, version)
);

CREATE TABLE IF NOT EXISTS logs (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	level      TEXT NOT NULL,
	message    TEXT NOT NULL,
	detail     TEXT,
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS config (
	key        TEXT PRIMARY KEY,
	value      TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cache_stats (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	hour       TEXT NOT NULL UNIQUE,
	hits       INTEGER NOT NULL DEFAULT 0,
	misses     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_plugins_repo     ON plugins(repo_id);
CREATE INDEX IF NOT EXISTS idx_versions_plugin  ON plugin_versions(plugin_id);
CREATE INDEX IF NOT EXISTS idx_versions_status  ON plugin_versions(download_status);
CREATE INDEX IF NOT EXISTS idx_logs_created     ON logs(created_at);
`
