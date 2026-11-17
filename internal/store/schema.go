package store

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL DEFAULT '',
	password_hash TEXT NOT NULL DEFAULT '',
	admin INTEGER NOT NULL DEFAULT 0,
	avatar_url TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	user_id INTEGER NOT NULL REFERENCES users(id),
	token_hash TEXT NOT NULL UNIQUE,
	csrf_token TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_exp ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS orgs (
	id TEXT PRIMARY KEY,
	user_id INTEGER NOT NULL REFERENCES users(id),
	name TEXT NOT NULL,
	kind TEXT NOT NULL DEFAULT 'personal',
	tier TEXT NOT NULL DEFAULT 'free',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	org_id TEXT NOT NULL REFERENCES orgs(id),
	name TEXT NOT NULL,
	prefix TEXT NOT NULL,
	key_hash TEXT NOT NULL UNIQUE,
	scopes TEXT NOT NULL DEFAULT '[]',
	expires_at TEXT,
	last_used_at TEXT,
	created_at TEXT NOT NULL,
	revoked_at TEXT
);

CREATE TABLE IF NOT EXISTS machines (
	id TEXT PRIMARY KEY,
	org_id TEXT NOT NULL REFERENCES orgs(id),
	name TEXT NOT NULL,
	desired_state TEXT NOT NULL,
	state TEXT NOT NULL,
	revision INTEGER NOT NULL DEFAULT 1,
	operation_id TEXT,
	node_id TEXT,
	template_id TEXT NOT NULL DEFAULT '',
	template_version TEXT,
	template_digest TEXT,
	snapshot_id TEXT,
	vcpus INTEGER NOT NULL DEFAULT 4,
	mem_mib INTEGER NOT NULL DEFAULT 8192,
	root_disk_gib INTEGER NOT NULL DEFAULT 16,
	persistent_disk_gib INTEGER NOT NULL DEFAULT 20,
	guest_user TEXT NOT NULL DEFAULT 'user',
	env_names TEXT NOT NULL DEFAULT '[]',
	env_blob BLOB NOT NULL DEFAULT '',
	auto_suspend_after_seconds INTEGER NOT NULL DEFAULT 1800,
	max_run_minutes INTEGER NOT NULL DEFAULT 120,
	run_started_at TEXT,
	ready_at TEXT,
	cooldown_until TEXT,
	auto_suspended_at TEXT,
	storage_synced INTEGER NOT NULL DEFAULT 0,
	snapshot_seq INTEGER NOT NULL DEFAULT 0,
	metadata TEXT NOT NULL DEFAULT '{}',
	vault INTEGER NOT NULL DEFAULT 1,
	free INTEGER NOT NULL DEFAULT 1,
	oom_kills INTEGER NOT NULL DEFAULT 0,
	archive_at TEXT,
	delete_at TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	UNIQUE(org_id, name)
);
CREATE INDEX IF NOT EXISTS idx_machines_org_state ON machines(org_id, state);
CREATE INDEX IF NOT EXISTS idx_machines_delete_at ON machines(delete_at);
CREATE INDEX IF NOT EXISTS idx_machines_archive_at ON machines(archive_at);

CREATE TABLE IF NOT EXISTS operations (
	id TEXT PRIMARY KEY,
	machine_id TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
	kind TEXT NOT NULL,
	status TEXT NOT NULL,
	error TEXT,
	result TEXT,
	request TEXT NOT NULL DEFAULT '{}',
	idempotency_key TEXT UNIQUE,
	started_at TEXT,
	finished_at TEXT,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_operations_machine ON operations(machine_id, created_at);
CREATE INDEX IF NOT EXISTS idx_operations_status ON operations(status);

CREATE TABLE IF NOT EXISTS snapshots (
	id TEXT PRIMARY KEY,
	org_id TEXT NOT NULL REFERENCES orgs(id),
	name TEXT NOT NULL,
	source_machine_id TEXT NOT NULL,
	source_snapshot_seq INTEGER NOT NULL DEFAULT 0,
	template_id TEXT NOT NULL DEFAULT '',
	template_version TEXT NOT NULL DEFAULT '',
	template_digest TEXT NOT NULL DEFAULT '',
	root_digest TEXT NOT NULL DEFAULT '',
	data_digest TEXT NOT NULL DEFAULT '',
	root_size_bytes INTEGER NOT NULL DEFAULT 0,
	data_size_bytes INTEGER NOT NULL DEFAULT 0,
	default_spec TEXT NOT NULL DEFAULT '{}',
	status TEXT NOT NULL DEFAULT 'saving',
	root_only INTEGER NOT NULL DEFAULT 0,
	guest_user TEXT NOT NULL DEFAULT 'user',
	created_at TEXT NOT NULL,
	UNIQUE(org_id, name)
);

CREATE TABLE IF NOT EXISTS ports (
	id TEXT PRIMARY KEY,
	machine_id TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
	port INTEGER NOT NULL,
	name TEXT NOT NULL DEFAULT '',
	protocol TEXT NOT NULL DEFAULT 'http',
	access TEXT NOT NULL DEFAULT 'public',
	token_hash TEXT,
	created_at TEXT NOT NULL,
	expires_at TEXT,
	UNIQUE(machine_id, port)
);

CREATE TABLE IF NOT EXISTS secrets (
	org_id TEXT NOT NULL,
	name TEXT NOT NULL,
	allowed_hosts TEXT NOT NULL DEFAULT '[]',
	value_ciphertext BLOB NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (org_id, name)
);

CREATE TABLE IF NOT EXISTS metrics (
	machine_id TEXT NOT NULL,
	sampled_at TEXT NOT NULL,
	cpu_pct REAL NOT NULL DEFAULT 0,
	mem_used_bytes INTEGER NOT NULL DEFAULT 0,
	mem_total_bytes INTEGER NOT NULL DEFAULT 0,
	net_rx_bytes INTEGER NOT NULL DEFAULT 0,
	net_tx_bytes INTEGER NOT NULL DEFAULT 0,
	disk_r_bytes INTEGER NOT NULL DEFAULT 0,
	disk_w_bytes INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (machine_id, sampled_at)
);

CREATE TABLE IF NOT EXISTS exec_history (
	id TEXT PRIMARY KEY,
	machine_id TEXT NOT NULL,
	cmd TEXT NOT NULL,
	cwd TEXT NOT NULL DEFAULT '',
	exit_code INTEGER NOT NULL DEFAULT -1,
	stdout_head BLOB,
	stderr_head BLOB,
	started_at TEXT NOT NULL,
	finished_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_exec_history_machine ON exec_history(machine_id, started_at);

CREATE TABLE IF NOT EXISTS activity_events (
	machine_id TEXT NOT NULL,
	kind TEXT NOT NULL,
	at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_activity_machine_at ON activity_events(machine_id, at);

CREATE TABLE IF NOT EXISTS webhooks (
	id TEXT PRIMARY KEY,
	org_id TEXT NOT NULL,
	url TEXT NOT NULL,
	secret TEXT NOT NULL DEFAULT '',
	events TEXT NOT NULL DEFAULT '[]',
	created_at TEXT NOT NULL
);
`
