// Package store implements the SQLite persistence layer for the control plane.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ibox/ibox/internal/model"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // serialize writes; WAL allows concurrent reads
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(schema)
	return err
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// ---- helpers ----

func execJSON(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ---- users / sessions ----

func (s *Store) UpsertUser(email, name, passwordHash string, admin bool) (int, error) {
	res, err := s.db.Exec(`INSERT INTO users(email,name,password_hash,admin,created_at)
		VALUES(?,?,?,?,?)
		ON CONFLICT(email) DO UPDATE SET name=excluded.name, admin=excluded.admin`,
		email, name, passwordHash, admin, now())
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err == nil && id > 0 {
		return int(id), nil
	}
	var uid int
	err = s.db.QueryRow(`SELECT id FROM users WHERE email=?`, email).Scan(&uid)
	return uid, err
}

type UserRow struct {
	ID           int
	Email        string
	Name         string
	PasswordHash string
	Admin        bool
}

func (s *Store) GetUserByEmail(email string) (*UserRow, error) {
	var u UserRow
	err := s.db.QueryRow(`SELECT id,email,name,password_hash,admin FROM users WHERE email=?`, email).
		Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Admin)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(uid int) (*UserRow, error) {
	var u UserRow
	err := s.db.QueryRow(`SELECT id,email,name,password_hash,admin FROM users WHERE id=?`, uid).
		Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Admin)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) CreateSession(userID int, tokenHash, csrf string, ttl time.Duration) (string, error) {
	sid := fmt.Sprintf("ses_%d_%d", userID, time.Now().UnixNano())
	_, err := s.db.Exec(`INSERT INTO sessions(id,user_id,token_hash,csrf_token,expires_at,created_at)
		VALUES(?,?,?,?,?,?)`, sid, userID, tokenHash, csrf, time.Now().UTC().Add(ttl).Format(time.RFC3339Nano), now())
	return sid, err
}

type SessionRow struct {
	ID        string
	UserID    int
	CSRFToken string
	ExpiresAt time.Time
}

func (s *Store) GetSessionByTokenHash(hash string) (*SessionRow, error) {
	var r SessionRow
	var exp string
	err := s.db.QueryRow(`SELECT id,user_id,csrf_token,expires_at FROM sessions WHERE token_hash=?`, hash).
		Scan(&r.ID, &r.UserID, &r.CSRFToken, &exp)
	if err != nil {
		return nil, err
	}
	r.ExpiresAt, err = time.Parse(time.RFC3339Nano, exp)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(r.ExpiresAt) {
		return nil, sql.ErrNoRows
	}
	return &r, nil
}

func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id=?`, id)
	return err
}

// ---- orgs ----

func (s *Store) ListOrgIDs() ([]string, error) {
	rows, err := s.db.Query(`SELECT id FROM orgs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CountActivitySince counts activity events of a kind since t.
func (s *Store) CountActivitySince(machineID string, since time.Time, kind string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM activity_events WHERE machine_id=? AND kind=? AND at>=?`,
		machineID, kind, since.UTC().Format(time.RFC3339Nano)).Scan(&n)
	return n, err
}

func (s *Store) EnsurePersonalOrg(userID int, email string) (string, error) {
	var oid string
	err := s.db.QueryRow(`SELECT id FROM orgs WHERE user_id=? AND kind='personal'`, userID).Scan(&oid)
	if err == nil {
		return oid, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	oid = newOrgID()
	_, err = s.db.Exec(`INSERT INTO orgs(id,user_id,name,kind,tier,created_at) VALUES(?,?,?,?,?,?)`,
		oid, userID, email, "personal", "free", now())
	return oid, err
}

func (s *Store) GetOrgByUser(userID int) (*model.Org, error) {
	var o model.Org
	err := s.db.QueryRow(`SELECT id,name,tier,kind,created_at FROM orgs WHERE user_id=? AND kind='personal'`, userID).
		Scan(&o.ID, &o.Name, &o.Tier, &o.Kind, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	o.Personal = o.Kind == "personal"
	o.Role = "owner"
	return &o, nil
}

func newOrgID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("org_%016x%016x", time.Now().UnixNano(), b)
}

// ---- machines ----

func (s *Store) InsertMachine(m *model.Machine) error {
	envNames := execJSON(m.EnvNames)
	meta := execJSON(m.Metadata)
	var tmplID, tmplVersion, tmplDigest any
	if m.Template != nil {
		tmplID = m.Template.ID
		tmplVersion = m.Template.Version
		tmplDigest = m.Template.Digest
	}
	var snapID any
	if m.Snapshot != nil {
		snapID = m.Snapshot.ID
	}
	_, err := s.db.Exec(`INSERT INTO machines
		(id,org_id,name,desired_state,state,revision,operation_id,node_id,
		 template_id,template_version,template_digest,snapshot_id,
		 vcpus,mem_mib,root_disk_gib,persistent_disk_gib,
		 guest_user,env_names,env_blob,auto_suspend_after_seconds,max_run_minutes,
		 run_started_at,ready_at,cooldown_until,auto_suspended_at,
		 storage_synced,snapshot_seq,metadata,vault,free,oom_kills,
		 archive_at,delete_at,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.OrgID, m.Name, m.DesiredState, m.State, m.Revision, nilIfEmpty(m.OperationID), nilIfEmpty(m.NodeID),
		tmplID, tmplVersion, tmplDigest, snapID,
		m.Spec.VCPUs, m.Spec.MemMiB, m.Spec.RootDiskGiB, m.Spec.PersistentDiskGiB,
		m.GuestUser, envNames, []byte{}, m.AutoSuspendAfterSeconds, m.MaxRunMinutes,
		m.RunStartedAt, m.ReadyAt, m.CooldownUntil, m.AutoSuspendedAt,
		boolToInt(m.Storage.Synced), m.Storage.SnapshotSeq, meta, boolToInt(m.Vault), boolToInt(m.Free), m.OOMKills,
		m.ArchiveAt, m.DeleteAt, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *Store) scanMachine(row interface{ Scan(...any) error }) (*model.Machine, error) {
	var m model.Machine
	var tmplID, tmplDigest, tmplVersion *string
	var snapID *string
	var nodeID, opID *string
	var runStartedAt, readyAt, cooldownUntil, autoSuspendedAt, archiveAt, deleteAt *string
	var envNames, meta string
	var envBlob []byte
	var synced, vault, free int
	err := row.Scan(
		&m.ID, &m.OrgID, &m.Name, &m.DesiredState, &m.State, &m.Revision, &opID, &nodeID,
		&tmplID, &tmplVersion, &tmplDigest, &snapID,
		&m.Spec.VCPUs, &m.Spec.MemMiB, &m.Spec.RootDiskGiB, &m.Spec.PersistentDiskGiB,
		&m.GuestUser, &envNames, &envBlob, &m.AutoSuspendAfterSeconds, &m.MaxRunMinutes,
		&runStartedAt, &readyAt, &cooldownUntil, &autoSuspendedAt,
		&synced, &m.Storage.SnapshotSeq, &meta, &vault, &free, &m.OOMKills,
		&archiveAt, &deleteAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.OperationID = deref(opID)
	m.NodeID = deref(nodeID)
	if tmplID != nil {
		m.Template = &model.TemplateRef{ID: *tmplID, Version: deref(tmplVersion), Digest: deref(tmplDigest)}
	}
	if snapID != nil {
		m.Snapshot = &model.SnapshotRef{ID: *snapID}
		m.Image = m.Snapshot
	}
	m.RunStartedAt = deref(runStartedAt)
	m.ReadyAt = deref(readyAt)
	m.CooldownUntil = deref(cooldownUntil)
	m.AutoSuspendedAt = deref(autoSuspendedAt)
	m.ArchiveAt = deref(archiveAt)
	m.DeleteAt = deref(deleteAt)
	m.Storage.SizeGiB = m.Spec.PersistentDiskGiB
	m.Storage.MountPath = "/data"
	m.Storage.Persistent = true
	m.Storage.Synced = synced == 1
	m.Vault = vault == 1
	m.Free = free == 1
	m.Runtime.ProcessesPreserved = m.State == model.StateRunning || m.State == model.StateSuspended
	m.Runtime.Ready = m.State == model.StateRunning
	_ = json.Unmarshal([]byte(envNames), &m.EnvNames)
	_ = json.Unmarshal([]byte(meta), &m.Metadata)
	if m.Metadata == nil {
		m.Metadata = map[string]string{}
	}
	return &m, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

const machineCols = `id,org_id,name,desired_state,state,revision,operation_id,node_id,
	template_id,template_version,template_digest,snapshot_id,
	vcpus,mem_mib,root_disk_gib,persistent_disk_gib,
	guest_user,env_names,env_blob,auto_suspend_after_seconds,max_run_minutes,
	run_started_at,ready_at,cooldown_until,auto_suspended_at,
	storage_synced,snapshot_seq,metadata,vault,free,oom_kills,
	archive_at,delete_at,created_at,updated_at`

func (s *Store) GetMachine(idOrName, orgID string) (*model.Machine, error) {
	row := s.db.QueryRow(`SELECT `+machineCols+` FROM machines WHERE org_id=? AND (id=? OR name=?)`, orgID, idOrName, idOrName)
	return s.scanMachine(row)
}

func (s *Store) GetMachineByID(id string) (*model.Machine, error) {
	row := s.db.QueryRow(`SELECT `+machineCols+` FROM machines WHERE id=?`, id)
	return s.scanMachine(row)
}

func (s *Store) ListMachines(orgID string) ([]*model.Machine, error) {
	rows, err := s.db.Query(`SELECT `+machineCols+` FROM machines WHERE org_id=? ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.Machine{}
	for rows.Next() {
		m, err := s.scanMachine(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) CountMachines(orgID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM machines WHERE org_id=? AND state!=?`, orgID, model.StateDeleting).Scan(&n)
	return n, err
}

func (s *Store) CountRunning(orgID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM machines WHERE org_id=? AND state IN ('running','starting','resuming')`, orgID).Scan(&n)
	return n, err
}

// UpdateMachine applies a mutation; bumps revision and updated_at.
func (s *Store) UpdateMachine(id string, fn func(m *model.Machine)) error {
	for {
		m, err := s.GetMachineByID(id)
		if err != nil {
			return err
		}
		fn(m)
		res, err := s.db.Exec(`UPDATE machines SET desired_state=?,state=?,revision=?,operation_id=?,
			vcpus=?,mem_mib=?,root_disk_gib=?,persistent_disk_gib=?,
			auto_suspend_after_seconds=?,max_run_minutes=?,run_started_at=?,ready_at=?,cooldown_until=?,auto_suspended_at=?,
			storage_synced=?,snapshot_seq=?,metadata=?,vault=?,free=?,oom_kills=?,node_id=?,
			archive_at=?,delete_at=?,updated_at=?
			WHERE id=? AND revision=?`,
			m.DesiredState, m.State, m.Revision+1, nilIfEmpty(m.OperationID),
			m.Spec.VCPUs, m.Spec.MemMiB, m.Spec.RootDiskGiB, m.Spec.PersistentDiskGiB,
			m.AutoSuspendAfterSeconds, m.MaxRunMinutes, nilIfEmpty(m.RunStartedAt), nilIfEmpty(m.ReadyAt), nilIfEmpty(m.CooldownUntil), nilIfEmpty(m.AutoSuspendedAt),
			boolToInt(m.Storage.Synced), m.Storage.SnapshotSeq, execJSON(m.Metadata), boolToInt(m.Vault), boolToInt(m.Free), m.OOMKills, nilIfEmpty(m.NodeID),
			nilIfEmpty(m.ArchiveAt), nilIfEmpty(m.DeleteAt), now(), id, m.Revision)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 1 {
			return nil
		}
		// optimistic lock conflict; retry
	}
}

func (s *Store) RenameMachine(id, name string) error {
	_, err := s.db.Exec(`UPDATE machines SET name=?, updated_at=? WHERE id=?`, name, now(), id)
	return err
}

func (s *Store) SetMachineEnv(id string, names []string, blob []byte) error {
	_, err := s.db.Exec(`UPDATE machines SET env_names=?, env_blob=?, updated_at=? WHERE id=?`, execJSON(names), blob, now(), id)
	return err
}

func (s *Store) GetMachineEnvBlob(id string) ([]byte, error) {
	var b []byte
	err := s.db.QueryRow(`SELECT env_blob FROM machines WHERE id=?`, id).Scan(&b)
	return b, err
}

func (s *Store) DeleteMachine(id string) error {
	// remove dependent rows first (FK safety on databases created before
	// ON DELETE CASCADE was added to the schema)
	for _, stmt := range []string{
		`DELETE FROM ports WHERE machine_id=?`,
		`DELETE FROM operations WHERE machine_id=?`,
		`DELETE FROM metrics WHERE machine_id=?`,
		`DELETE FROM exec_history WHERE machine_id=?`,
		`DELETE FROM activity_events WHERE machine_id=?`,
	} {
		if _, err := s.db.Exec(stmt, id); err != nil {
			return err
		}
	}
	if _, err := s.db.Exec(`DELETE FROM machines WHERE id=?`, id); err != nil {
		return err
	}
	return nil
}

// ---- operations ----

func (s *Store) InsertOperation(op *model.Operation) error {
	_, err := s.db.Exec(`INSERT INTO operations(id,machine_id,kind,status,error,result,request,idempotency_key,created_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		op.ID, op.MachineID, op.Kind, op.Status, nilIfEmpty(opError(op)), execJSON(op.Result), "{}", op.ID, now())
	return err
}

func opError(op *model.Operation) string {
	if op.Error == nil {
		return ""
	}
	b, _ := json.Marshal(op.Error)
	return string(b)
}

func (s *Store) GetOperation(id string) (*model.Operation, error) {
	var op model.Operation
	var errStr *string
	var startedAt, finishedAt *string
	err := s.db.QueryRow(`SELECT id,machine_id,kind,status,error,result,started_at,finished_at,created_at
		FROM operations WHERE id=?`, id).
		Scan(&op.ID, &op.MachineID, &op.Kind, &op.Status, &errStr, &op.Result, &startedAt, &finishedAt, &op.CreatedAt)
	if err != nil {
		return nil, err
	}
	if errStr != nil && *errStr != "" {
		var e model.Error
		if json.Unmarshal([]byte(*errStr), &e) == nil {
			op.Error = &e
		}
	}
	op.StartedAt = deref(startedAt)
	op.FinishedAt = deref(finishedAt)
	return &op, nil
}

func (s *Store) FindOperationByIdempotencyKey(key string) (*model.Operation, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM operations WHERE idempotency_key=?`, key).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetOperation(id)
}

// AcquireMachineLock sets the machine's current operation if none is running.
func (s *Store) AcquireMachineLock(machineID, opID string) (bool, error) {
	res, err := s.db.Exec(`UPDATE machines SET operation_id=?, updated_at=? WHERE id=? AND operation_id IS NULL`,
		opID, now(), machineID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (s *Store) ReleaseMachineLock(machineID string) error {
	_, err := s.db.Exec(`UPDATE machines SET operation_id=NULL, updated_at=? WHERE id=?`, now(), machineID)
	return err
}

func (s *Store) SetOperationRunning(id string) error {
	_, err := s.db.Exec(`UPDATE operations SET status='running', started_at=? WHERE id=?`, now(), id)
	return err
}

func (s *Store) FinishOperation(id, status string, opErr *model.Error, result any) error {
	_, err := s.db.Exec(`UPDATE operations SET status=?, error=?, result=?, finished_at=? WHERE id=?`,
		status, opError(&model.Operation{Error: opErr}), execJSON(result), now(), id)
	return err
}

func (s *Store) ListRecentOperations(machineID string, limit int) ([]*model.Operation, error) {
	rows, err := s.db.Query(`SELECT id,machine_id,kind,status,error,result,started_at,finished_at,created_at
		FROM operations WHERE machine_id=? ORDER BY created_at DESC LIMIT ?`, machineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Operation
	for rows.Next() {
		var op model.Operation
		var errStr *string
		var startedAt, finishedAt *string
		if err := rows.Scan(&op.ID, &op.MachineID, &op.Kind, &op.Status, &errStr, &op.Result, &startedAt, &finishedAt, &op.CreatedAt); err != nil {
			return nil, err
		}
		if errStr != nil && *errStr != "" {
			var e model.Error
			if json.Unmarshal([]byte(*errStr), &e) == nil {
				op.Error = &e
			}
		}
		op.StartedAt = deref(startedAt)
		op.FinishedAt = deref(finishedAt)
		out = append(out, &op)
	}
	return out, rows.Err()
}

// ---- snapshots ----

func (s *Store) InsertSnapshot(sn *model.Snapshot) error {
	_, err := s.db.Exec(`INSERT INTO snapshots
		(id,org_id,name,source_machine_id,source_snapshot_seq,template_id,template_version,template_digest,
		 root_digest,data_digest,root_size_bytes,data_size_bytes,default_spec,status,root_only,guest_user,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sn.ID, sn.OrgID, sn.Name, sn.SourceMachineID, sn.SourceSnapshotSeq,
		sn.Template.ID, sn.Template.Version, sn.Template.Digest,
		sn.RootDigest, sn.DataDigest, sn.RootSizeBytes, sn.DataSizeBytes,
		execJSON(sn.DefaultSpec), sn.Status, boolToInt(sn.RootOnly), sn.GuestUser, sn.CreatedAt)
	return err
}

func (s *Store) scanSnapshot(row interface{ Scan(...any) error }) (*model.Snapshot, error) {
	var sn model.Snapshot
	var specJSON string
	var rootOnly int
	err := row.Scan(&sn.ID, &sn.OrgID, &sn.Name, &sn.SourceMachineID, &sn.SourceSnapshotSeq,
		&sn.Template.ID, &sn.Template.Version, &sn.Template.Digest,
		&sn.RootDigest, &sn.DataDigest, &sn.RootSizeBytes, &sn.DataSizeBytes,
		&specJSON, &sn.Status, &rootOnly, &sn.GuestUser, &sn.CreatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(specJSON), &sn.DefaultSpec)
	sn.RootOnly = rootOnly == 1
	return &sn, nil
}

const snapshotCols = `id,org_id,name,source_machine_id,source_snapshot_seq,template_id,template_version,template_digest,
	root_digest,data_digest,root_size_bytes,data_size_bytes,default_spec,status,root_only,guest_user,created_at`

func (s *Store) GetSnapshot(idOrName, orgID string) (*model.Snapshot, error) {
	row := s.db.QueryRow(`SELECT `+snapshotCols+` FROM snapshots WHERE org_id=? AND (id=? OR name=?)`, orgID, idOrName, idOrName)
	return s.scanSnapshot(row)
}

func (s *Store) GetSnapshotByID(id string) (*model.Snapshot, error) {
	row := s.db.QueryRow(`SELECT `+snapshotCols+` FROM snapshots WHERE id=?`, id)
	return s.scanSnapshot(row)
}

func (s *Store) ListSnapshots(orgID string) ([]*model.Snapshot, error) {
	rows, err := s.db.Query(`SELECT `+snapshotCols+` FROM snapshots WHERE org_id=? ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Snapshot
	for rows.Next() {
		sn, err := s.scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sn)
	}
	return out, rows.Err()
}

func (s *Store) CountSnapshots(orgID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM snapshots WHERE org_id=?`, orgID).Scan(&n)
	return n, err
}

func (s *Store) UpdateSnapshotStatus(id, status string) error {
	_, err := s.db.Exec(`UPDATE snapshots SET status=? WHERE id=?`, status, id)
	return err
}

func (s *Store) DeleteSnapshot(id string) error {
	_, err := s.db.Exec(`DELETE FROM snapshots WHERE id=?`, id)
	return err
}

// ---- ports ----

// InsertPort inserts a port row; tokenHash is stored but never exposed.
func (s *Store) InsertPort(p *model.Port, tokenHash string) error {
	_, err := s.db.Exec(`INSERT INTO ports(id,machine_id,port,name,protocol,access,token_hash,created_at,expires_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		p.ID, p.MachineID, p.Port, p.Name, p.Protocol, p.Access, nilIfEmpty(tokenHash), p.CreatedAt, nilIfEmpty(p.ExpiresAt))
	return err
}

// SetPortTokenHash stores hash separately (not exposed in model JSON).
func (s *Store) SetPortTokenHash(portID, hash string) error {
	_, err := s.db.Exec(`UPDATE ports SET token_hash=? WHERE id=?`, hash, portID)
	return err
}

func (s *Store) GetPortTokenHash(portID string) (string, error) {
	var h *string
	err := s.db.QueryRow(`SELECT token_hash FROM ports WHERE id=?`, portID).Scan(&h)
	if err != nil {
		return "", err
	}
	return deref(h), nil
}

func (s *Store) scanPort(row interface{ Scan(...any) error }) (*model.Port, error) {
	var p model.Port
	var expiresAt *string
	err := row.Scan(&p.ID, &p.MachineID, &p.Port, &p.Name, &p.Protocol, &p.Access, &expiresAt, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.ExpiresAt = deref(expiresAt)
	return &p, nil
}

func (s *Store) GetPort(machineID string, port int) (*model.Port, error) {
	row := s.db.QueryRow(`SELECT id,machine_id,port,name,protocol,access,expires_at,created_at FROM ports WHERE machine_id=? AND port=?`, machineID, port)
	return s.scanPort(row)
}

func (s *Store) GetPortByID(id string) (*model.Port, error) {
	row := s.db.QueryRow(`SELECT id,machine_id,port,name,protocol,access,expires_at,created_at FROM ports WHERE id=?`, id)
	return s.scanPort(row)
}

func (s *Store) ListPortsForMachine(machineID string) ([]*model.Port, error) {
	rows, err := s.db.Query(`SELECT id,machine_id,port,name,protocol,access,expires_at,created_at FROM ports WHERE machine_id=? ORDER BY port`, machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Port
	for rows.Next() {
		p, err := s.scanPort(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListPorts(orgID string) ([]*model.Port, error) {
	rows, err := s.db.Query(`SELECT p.id,p.machine_id,p.port,p.name,p.protocol,p.access,p.expires_at,p.created_at
		FROM ports p JOIN machines m ON p.machine_id=m.id WHERE m.org_id=? ORDER BY p.created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Port
	for rows.Next() {
		p, err := s.scanPort(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) UpdatePort(portID, name, access string) error {
	_, err := s.db.Exec(`UPDATE ports SET name=?, access=? WHERE id=?`, name, access, portID)
	return err
}

func (s *Store) DeletePort(machineID string, port int) error {
	_, err := s.db.Exec(`DELETE FROM ports WHERE machine_id=? AND port=?`, machineID, port)
	return err
}

func (s *Store) CountPortsForMachine(machineID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM ports WHERE machine_id=?`, machineID).Scan(&n)
	return n, err
}

// ---- metrics ----

func (s *Store) InsertMetric(machineID string, smp *model.MetricSample) error {
	_, err := s.db.Exec(`INSERT INTO metrics(machine_id,sampled_at,cpu_pct,mem_used_bytes,mem_total_bytes,net_rx_bytes,net_tx_bytes,disk_r_bytes,disk_w_bytes)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		machineID, smp.At.UTC().Format(time.RFC3339Nano), smp.CPUPercent, smp.MemUsedBytes, smp.MemTotalBytes,
		smp.NetRxBytes, smp.NetTxBytes, s_mp(smp.DiskRBytes), s_mp(smp.DiskWBytes))
	return err
}

func s_mp(v int64) int64 { return v }

func (s *Store) ListMetrics(machineID string, since time.Time) ([]*model.MetricSample, error) {
	rows, err := s.db.Query(`SELECT sampled_at,cpu_pct,mem_used_bytes,mem_total_bytes,net_rx_bytes,net_tx_bytes,disk_r_bytes,disk_w_bytes
		FROM metrics WHERE machine_id=? AND sampled_at>=? ORDER BY sampled_at`, machineID, since.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.MetricSample{}
	for rows.Next() {
		var smp model.MetricSample
		var at string
		if err := rows.Scan(&at, &smp.CPUPercent, &smp.MemUsedBytes, &smp.MemTotalBytes, &smp.NetRxBytes, &smp.NetTxBytes, &smp.DiskRBytes, &smp.DiskWBytes); err != nil {
			return nil, err
		}
		smp.At, _ = time.Parse(time.RFC3339Nano, at)
		out = append(out, &smp)
	}
	return out, rows.Err()
}

// ---- exec history ----

func (s *Store) InsertExecHistory(machineID, cmd, cwd string, exitCode int, stdout, stderr []byte, startedAt, finishedAt string) (string, error) {
	eid := fmt.Sprintf("ex_%d", time.Now().UnixNano())
	if len(stdout) > 64*1024 {
		stdout = stdout[:64*1024]
	}
	if len(stderr) > 64*1024 {
		stderr = stderr[:64*1024]
	}
	_, err := s.db.Exec(`INSERT INTO exec_history(id,machine_id,cmd,cwd,exit_code,stdout_head,stderr_head,started_at,finished_at)
		VALUES(?,?,?,?,?,?,?,?,?)`, eid, machineID, cmd, cwd, exitCode, stdout, stderr, startedAt, finishedAt)
	return eid, err
}

func (s *Store) ListExecHistory(machineID string, limit int) ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT id,cmd,cwd,exit_code,started_at,finished_at FROM exec_history
		WHERE machine_id=? ORDER BY started_at DESC LIMIT ?`, machineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, cmd, cwd string
		var exitCode int
		var startedAt, finishedAt string
		if err := rows.Scan(&id, &cmd, &cwd, &exitCode, &startedAt, &finishedAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "cmd": cmd, "cwd": cwd, "exitCode": exitCode, "startedAt": startedAt, "finishedAt": finishedAt})
	}
	return out, rows.Err()
}

// ---- activity ----

func (s *Store) RecordActivity(machineID, kind string) error {
	_, err := s.db.Exec(`INSERT INTO activity_events(machine_id,kind,at) VALUES(?,?,?)`,
		machineID, kind, now())
	return err
}

func (s *Store) LatestActivity(machineID string) (time.Time, error) {
	var at string
	err := s.db.QueryRow(`SELECT at FROM activity_events WHERE machine_id=? ORDER BY at DESC LIMIT 1`, machineID).Scan(&at)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, at)
}

// ---- api keys ----

func (s *Store) InsertAPIKey(orgID, name, prefix, hash string, scopes []string, expiresAt string) (string, error) {
	kid := fmt.Sprintf("key_%d", time.Now().UnixNano())
	_, err := s.db.Exec(`INSERT INTO api_keys(id,org_id,name,prefix,key_hash,scopes,expires_at,created_at)
		VALUES(?,?,?,?,?,?,?,?)`, kid, orgID, name, prefix, hash, execJSON(scopes), nilIfEmpty(expiresAt), now())
	return kid, err
}

type APIKeyRow struct {
	ID        string
	OrgID     string
	Name      string
	Prefix    string
	KeyHash   string
	Scopes    []string
	ExpiresAt string
	RevokedAt string
}

func (s *Store) GetAPIKeyByHash(hash string) (*APIKeyRow, error) {
	var k APIKeyRow
	var scopes string
	var expiresAt, revokedAt *string
	err := s.db.QueryRow(`SELECT id,org_id,name,prefix,key_hash,scopes,expires_at,revoked_at FROM api_keys WHERE key_hash=?`, hash).
		Scan(&k.ID, &k.OrgID, &k.Name, &k.Prefix, &k.KeyHash, &scopes, &expiresAt, &revokedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(scopes), &k.Scopes)
	k.ExpiresAt = deref(expiresAt)
	k.RevokedAt = deref(revokedAt)
	return &k, nil
}

func (s *Store) ListAPIKeys(orgID string) ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT id,name,prefix,scopes,expires_at,last_used_at,created_at,revoked_at FROM api_keys WHERE org_id=? ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, name, prefix, scopes, createdAt string
		var expiresAt, lastUsedAt, revokedAt *string
		if err := rows.Scan(&id, &name, &prefix, &scopes, &expiresAt, &lastUsedAt, &createdAt, &revokedAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "name": name, "prefix": prefix, "scopes": json.RawMessage(scopes),
			"expiresAt": deref(expiresAt), "lastUsedAt": deref(lastUsedAt), "createdAt": createdAt,
			"revokedAt": deref(revokedAt),
		})
	}
	return out, rows.Err()
}

func (s *Store) TouchAPIKey(id string) {
	_, _ = s.db.Exec(`UPDATE api_keys SET last_used_at=? WHERE id=?`, now(), id)
}

func (s *Store) RevokeAPIKey(id, orgID string) error {
	_, err := s.db.Exec(`UPDATE api_keys SET revoked_at=? WHERE id=? AND org_id=?`, now(), id, orgID)
	return err
}

// ---- secrets ----

func (s *Store) PutSecret(orgID, name string, allowedHosts []string, valueCiphertext []byte) error {
	_, err := s.db.Exec(`INSERT INTO secrets(org_id,name,allowed_hosts,value_ciphertext,created_at,updated_at)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT(org_id,name) DO UPDATE SET allowed_hosts=excluded.allowed_hosts, value_ciphertext=excluded.value_ciphertext, updated_at=excluded.updated_at`,
		orgID, name, execJSON(allowedHosts), valueCiphertext, now(), now())
	return err
}

func (s *Store) ListSecrets(orgID string) ([]map[string]any, error) {
	rows, err := s.db.Query(`SELECT name,allowed_hosts,created_at,updated_at FROM secrets WHERE org_id=? ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var name, hosts, createdAt, updatedAt string
		if err := rows.Scan(&name, &hosts, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"name": name, "allowedHosts": json.RawMessage(hosts), "createdAt": createdAt, "updatedAt": updatedAt})
	}
	return out, rows.Err()
}

func (s *Store) CountSecrets(orgID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM secrets WHERE org_id=?`, orgID).Scan(&n)
	return n, err
}

func (s *Store) DeleteSecret(orgID, name string) error {
	_, err := s.db.Exec(`DELETE FROM secrets WHERE org_id=? AND name=?`, orgID, name)
	return err
}

// ---- storage sums ----

func (s *Store) StorageBytes(orgID string) (logical int64, err error) {
	err = s.db.QueryRow(`SELECT COALESCE(SUM(size_bytes),0) FROM (
		SELECT (root_size_bytes + data_size_bytes) AS size_bytes FROM snapshots WHERE org_id=?)`, orgID).Scan(&logical)
	return
}
