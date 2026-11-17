// Package sandbox implements OCI-sandbox lifecycle over runsc/runc:
// create, start, suspend (checkpoint or freezer), resume, stop, fork, delete.
package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ibox/ibox/internal/model"
)

const (
	RuntimeRunsc = "runsc"
	RuntimeRunc  = "runc"
)

type State string

const (
	StateRunning   State = "running"
	StatePaused    State = "paused"
	StateStopped   State = "stopped"
	StateSuspended State = "suspended"
	StateNone      State = "none"
)

// Sandbox is one machine's runtime on this node.
type Sandbox struct {
	MachineID string
	Name      string
	Dir       string // /var/lib/ibox-agent/machines/<id>
	RootDir   string // rootfs dir (bind-mounted)
	DataDir   string // /data dir (bind-mounted)
	Runtime   string // runsc | runc
	Network string // host | none
	State   State

	PausedWithCheckpoint bool
	cgroupDir            string // cached cgroup v2 path

	mu sync.Mutex
	// port forwards: sandbox port -> host listener
	forwards map[int]*forward
	lastCPUTime uint64
	lastIO      ioCounters
	lastSample  time.Time
}

type ioCounters struct {
	rbytes, wbytes uint64
}

type Manager struct {
	BaseDir   string // /var/lib/ibox-agent
	Runtime   string // default runtime
	Network string // host | none
	CgroupRoot string

	mu       sync.Mutex
	sandboxes map[string]*Sandbox
}

func NewManager(baseDir, runtime, network string) *Manager {
	if runtime == "" {
		runtime = RuntimeRunsc
	}
	if network == "" {
		network = "host"
	}
	os.MkdirAll(filepath.Join(baseDir, "machines"), 0o755)
	return &Manager{
		BaseDir: baseDir, Runtime: runtime, Network: network,
		CgroupRoot: "/sys/fs/cgroup/ibox",
		sandboxes:  map[string]*Sandbox{},
	}
}

func (m *Manager) machineDir(id string) string {
	return filepath.Join(m.BaseDir, "machines", id)
}

// MachineTmpDir returns a temp working dir for a machine operation.
func (m *Manager) MachineTmpDir(id string) string {
	return filepath.Join(m.BaseDir, "tmp", id)
}

// TemplatesDir returns the shared template CAS root.
func (m *Manager) TemplatesDir() string {
	return filepath.Join(m.BaseDir, "rootfs")
}

// List returns all known sandboxes (loads persisted state from disk).
func (m *Manager) List() []*Sandbox {
	m.mu.Lock()
	defer m.mu.Unlock()
	entries, _ := os.ReadDir(filepath.Join(m.BaseDir, "machines"))
	for _, e := range entries {
		id := e.Name()
		if _, ok := m.sandboxes[id]; ok {
			continue
		}
		if sb := m.load(id); sb != nil {
			m.sandboxes[id] = sb
		}
	}
	out := make([]*Sandbox, 0, len(m.sandboxes))
	for _, sb := range m.sandboxes {
		out = append(out, sb)
	}
	return out
}

func (m *Manager) Get(id string) (*Sandbox, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sb, ok := m.sandboxes[id]; ok {
		return sb, true
	}
	if sb := m.load(id); sb != nil {
		m.sandboxes[id] = sb
		return sb, true
	}
	return nil, false
}

// load reads machine state from disk (survives agent restarts).
func (m *Manager) load(id string) *Sandbox {
	dir := m.machineDir(id)
	metaPath := filepath.Join(dir, "meta.json")
	b, err := os.ReadFile(metaPath)
	if err != nil {
		return nil
	}
	var meta struct {
		Name    string `json:"name"`
		Runtime string `json:"runtime"`
		State   string `json:"state"`
		Network string `json:"network"`
	}
	if json.Unmarshal(b, &meta) != nil {
		return nil
	}
	sb := &Sandbox{
		MachineID: id, Name: meta.Name, Dir: dir,
		RootDir: filepath.Join(dir, "root"), DataDir: filepath.Join(dir, "data"),
		Runtime: meta.Runtime, Network: meta.Network,
		State:   State(meta.State), forwards: map[int]*forward{},
	}
	if sb.State == StateRunning || sb.State == StatePaused {
		// verify the process is actually alive
		if !m.processAlive(id) {
			sb.State = StateStopped
			m.save(sb)
		}
	}
	return sb
}

func (m *Manager) save(sb *Sandbox) {
	meta := struct {
		Name    string `json:"name"`
		Runtime string `json:"runtime"`
		State   string `json:"state"`
		Network string `json:"network"`
	}{sb.Name, sb.Runtime, string(sb.State), sb.Network}
	b, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(sb.Dir, "meta.json"), b, 0o644)
}

func (m *Manager) processAlive(id string) bool {
	// runsc list / runc list
	runtime := m.Runtime
	out, err := exec.Command(runtime, "--root", m.runscRoot(runtime), "list", "-format=json").Output()
	if err != nil {
		return false
	}
	var items []struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(out, &items) != nil {
		return false
	}
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}

func (m *Manager) runscRoot(runtime string) string {
	return filepath.Join(m.BaseDir, "runsc-root", runtime)
}

// Spec for create/restore.
type CreateSpec struct {
	MachineID   string
	Name        string
	TemplateDir string   // source rootfs (template CAS); empty when restoring from snapshot dir
	SnapshotDir string   // explicit root dir to copy from (snapshot restore / fork source handled by caller)
	DataSource  string   // explicit data dir to copy from
	VCPUs       int
	MemMiB      int
	DiskGiB     int
	RootDiskGiB int
	GuestUser   string
	Env         map[string]string
	Runtime     string
	Network     string
}

// Create provisions disk state and starts the sandbox.
func (m *Manager) Create(spec CreateSpec) (*Sandbox, error) {
	id := spec.MachineID
	if id == "" {
		return nil, errors.New("machineId is required")
	}
	runtime := spec.Runtime
	if runtime == "" {
		runtime = m.Runtime
	}
	network := spec.Network
	if network == "" {
		network = m.Network
	}
	dir := m.machineDir(id)
	if _, err := os.Stat(filepath.Join(dir, "meta.json")); err == nil {
		return nil, errors.New("machine already exists")
	}
	// a leftover dir without meta.json means a previous create crashed or its
	// result was lost — clean it and start over
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	sb := &Sandbox{
		MachineID: id, Name: spec.Name, Dir: dir,
		RootDir: filepath.Join(dir, "root"), DataDir: filepath.Join(dir, "data"),
		Runtime: runtime, Network: network, State: StateStopped,
		forwards: map[int]*forward{},
	}
	if sb.Name == "" {
		sb.Name = id
	}
	// rootfs
	src := spec.SnapshotDir
	if src == "" {
		src = spec.TemplateDir
	}
	if src == "" {
		return nil, errors.New("no rootfs source")
	}
	if err := CopyDir(src, sb.RootDir); err != nil {
		return nil, fmt.Errorf("copy rootfs: %w", err)
	}
	// data volume — owned by the guest user so uid 1000 can write
	if spec.DataSource != "" {
		if err := CopyDir(spec.DataSource, sb.DataDir); err != nil {
			return nil, fmt.Errorf("copy data: %w", err)
		}
	} else {
		os.MkdirAll(filepath.Join(sb.DataDir, "home", "user"), 0o755)
	}
	ChownGuest(sb.DataDir)
	m.mu.Lock()
	m.sandboxes[sb.MachineID] = sb
	m.mu.Unlock()
	m.save(sb)
	return sb, nil
}

// ChownGuest recursively chowns a dir to the guest user (uid/gid 1000).
func ChownGuest(dir string) {
	_ = exec.Command("chown", "-R", "1000:1000", dir).Run()
}

// CopyDir copies src to dst using cp -a with reflink where possible.
func CopyDir(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return err
	}
	cmd := exec.Command("cp", "-a", "--reflink=auto", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// fallback: manual copy
		return manualCopy(src, dst)
	}
	_ = out
	return nil
}

func manualCopy(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, info.Mode())
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		default:
			in, err := os.Open(p)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
			if err != nil {
				return err
			}
			if _, err := ioCopy(out, in); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		}
	})
}

func ioCopy(dst ioWriter, src ioReader) (int64, error) {
	buf := make([]byte, 1<<20)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			wn, werr := dst.Write(buf[:n])
			total += int64(wn)
			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				return total, nil
			}
			return total, err
		}
	}
}

type ioWriter interface{ Write([]byte) (int, error) }
type ioReader interface{ Read([]byte) (int, error) }

// ociConfig builds the OCI config.json.
func (m *Manager) ociConfig(sb *Sandbox, spec *StartSpec) ([]byte, error) {
	if spec == nil {
		spec = &StartSpec{}
	}
	guest := spec.GuestUser
	if guest == "" {
		guest = "user"
	}
	env := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"HOME=/data/home/" + guest,
		"USER=" + guest,
		"HOSTNAME=" + sb.Name,
		"TERM=xterm-256color",
	}
	for k, v := range spec.Env {
		env = append(env, k+"="+v)
	}
	absRoot, _ := filepath.Abs(sb.RootDir)
	absData, _ := filepath.Abs(sb.DataDir)

	resources := map[string]any{}
	if spec.VCPUs > 0 {
		quota := 100000
		period := 100000
		// cpu.max: "$quota $period" per cpu → total = vcpus * 100000 per 100000us
		// cpu.cpus is a CPUSET LIST (e.g. "0-3"), not a count — a bare "4"
		// would pin the sandbox to a single host CPU.
		resources["cpu"] = map[string]any{
			"cpus":   fmt.Sprintf("0-%d", spec.VCPUs-1),
			"quota":  quota * spec.VCPUs,
			"period": period,
		}
	}
	if spec.MemMiB > 0 {
		resources["memory"] = map[string]any{"limit": spec.MemMiB << 20}
	}
	resources["pids"] = map[string]any{"limit": 1024}

	cfg := map[string]any{
		"ociVersion": "1.2.0",
		"process": map[string]any{
			"terminal": false,
			"user":     map[string]any{"uid": 1000, "gid": 1000}, // guest user (template: useradd user)
			"args":     spec.Entrypoint(),
			"env":      env,
			"cwd":      "/",
			"capabilities": map[string]any{
				"bounding":    []string{"CAP_AUDIT_WRITE", "CAP_KILL", "CAP_NET_BIND_SERVICE"},
				"effective":   []string{"CAP_AUDIT_WRITE", "CAP_KILL", "CAP_NET_BIND_SERVICE"},
				"permitted":   []string{"CAP_AUDIT_WRITE", "CAP_KILL", "CAP_NET_BIND_SERVICE"},
				"inheritable": []string{},
			},
			"noNewPrivileges": true,
		},
		"root": map[string]any{
			"path":     absRoot,
			"readonly": false,
		},
		"mounts": []map[string]any{
			{"destination": "/data", "source": absData, "type": "bind", "options": []string{"rbind", "rw"}},
			{"destination": "/dev", "type": "devpts"},
			{"destination": "/proc", "type": "proc"},
			{"destination": "/sys", "type": "sysfs"},
			{"destination": "/tmp", "type": "tmpfs"},
		},
		"linux": map[string]any{
			"namespaces": []map[string]any{
				{"type": "pid"},
				{"type": "ipc"},
				{"type": "uts"},
				{"type": "mount"},
			},
			"resources": resources,
		},
		"hostname": sb.Name,
	}
	if sb.Network == "host" {
		// no network namespace → share host net
	} else {
		cfg["linux"].(map[string]any)["namespaces"] = append(
			cfg["linux"].(map[string]any)["namespaces"].([]map[string]any),
			map[string]any{"type": "network"},
		)
	}
	return json.Marshal(cfg)
}

// StartSpec carries runtime parameters for start.
type StartSpec struct {
	VCPUs      int
	MemMiB     int
	GuestUser  string
	Env        map[string]string
	EntrypointOverride []string
	RestoreCheckpoint  bool
}

func (s *StartSpec) Entrypoint() []string {
	if len(s.EntrypointOverride) > 0 {
		return s.EntrypointOverride
	}
	// /sbin/init exists in Debian templates; fall back to a shell loop
	return []string{"/bin/sh", "-c", `if [ -x /sbin/init ]; then exec /sbin/init; elif [ -x /lib/systemd/systemd ]; then exec /lib/systemd/systemd; else exec /bin/sh -c 'while true; do sleep 3600; done'; fi`}
}

// Start boots a sandbox.
func (m *Manager) Start(sb *Sandbox, spec *StartSpec) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.State == StateRunning {
		return nil
	}
	if spec == nil {
		spec = &StartSpec{}
	}
	// checkpoint restore path
	restore := spec.RestoreCheckpoint
	ckpt := filepath.Join(sb.Dir, "checkpoint")
	if _, err := os.Stat(filepath.Join(ckpt, "state.json")); err == nil && !restore {
		// don't auto-restore on plain start; plain start = fresh boot
	}
	if restore {
		if _, err := os.Stat(filepath.Join(ckpt, "state.json")); err != nil {
			return errors.New("no checkpoint to restore")
		}
		return m.restoreLocked(sb, ckpt)
	}
	// fresh boot
	cfg, err := m.ociConfig(sb, spec)
	if err != nil {
		return err
	}
	bundle := filepath.Join(sb.Dir, "bundle")
	os.RemoveAll(bundle)
	os.MkdirAll(bundle, 0o755)
	cfgPath := filepath.Join(bundle, "config.json")
	if err := os.WriteFile(cfgPath, cfg, 0o644); err != nil {
		return err
	}
	root := filepath.Join(bundle, "rootfs")
	os.Remove(root)
	if err := os.Symlink(sb.RootDir, root); err != nil {
		return err
	}

	args := []string{"--root", m.runscRoot(sb.Runtime), "run", "--bundle", bundle, "--detach"}
	if sb.Runtime == RuntimeRunsc {
		if sb.Network == "host" {
			args = append([]string{"--network=host"}, args...)
		} else {
			args = append([]string{"--network=none"}, args...)
		}
		args = append(args, sb.MachineID)
	} else {
		args = append(args, sb.MachineID)
	}
	cmd := exec.Command(sb.Runtime, args...)
	// NB: write to a real file, not a pipe/bytes.Buffer — detached runsc keeps
	// stdout/stderr open in the sandbox/gofer children, which would block
	// CombinedOutput's copy goroutines forever (pipe never reaches EOF).
	logf, logErr := os.Create(filepath.Join(sb.Dir, "runsc.log"))
	if logErr != nil {
		return fmt.Errorf("start failed: %v", logErr)
	}
	cmd.Stdout = logf
	cmd.Stderr = logf
	err = cmd.Run()
	logf.Close()
	out, _ := os.ReadFile(filepath.Join(sb.Dir, "runsc.log"))
	if err != nil {
		if strings.Contains(string(out), "already exists") {
			// a previous start attempt created the container but its result was
			// lost (RPC timeout / agent restart) — adopt it if it is alive
			if m.processAlive(sb.MachineID) {
				sb.State = StateRunning
				sb.PausedWithCheckpoint = false
				m.save(sb)
				return nil
			}
			_ = exec.Command(sb.Runtime, "--root", m.runscRoot(sb.Runtime), "delete", "-force", sb.MachineID).Run()
		}
		log.Printf("start %s: %v: %s", sb.MachineID, err, out)
		return fmt.Errorf("start failed: %s", strings.TrimSpace(string(out)))
	}
	sb.State = StateRunning
	sb.PausedWithCheckpoint = false
	m.save(sb)
	return nil
}

func (m *Manager) restoreLocked(sb *Sandbox, ckpt string) error {
	bundle := filepath.Join(sb.Dir, "bundle")
	cfg, err := m.ociConfig(sb, &StartSpec{})
	if err != nil {
		return err
	}
	os.MkdirAll(bundle, 0o755)
	os.WriteFile(filepath.Join(bundle, "config.json"), cfg, 0o644)
	root := filepath.Join(bundle, "rootfs")
	os.Remove(root)
	os.Symlink(sb.RootDir, root)

	args := []string{
		"--root", m.runscRoot(sb.Runtime),
		"restore", "--bundle", bundle,
		"--image-path", ckpt,
		"--detach",
	}
	if sb.Runtime == RuntimeRunsc {
		if sb.Network == "host" {
			args = append([]string{"--network=host"}, args...)
		}
	}
	args = append(args, sb.MachineID)
	cmd := exec.Command(sb.Runtime, args...)
	// detached runsc keeps the pipes open in its children → CombinedOutput
	// would hang forever; write to a real file instead (same as Start).
	logf, logErr := os.Create(filepath.Join(sb.Dir, "runsc-restore.log"))
	if logErr != nil {
		return fmt.Errorf("restore failed: %v", logErr)
	}
	cmd.Stdout = logf
	cmd.Stderr = logf
	err = cmd.Run()
	logf.Close()
	out, _ := os.ReadFile(filepath.Join(sb.Dir, "runsc-restore.log"))
	if err != nil {
		log.Printf("restore %s failed: %v: %s", sb.MachineID, err, out)
		return fmt.Errorf("restore failed: %s", strings.TrimSpace(string(out)))
	}
	sb.State = StateRunning
	sb.PausedWithCheckpoint = true
	m.save(sb)
	return nil
}

// Suspend pauses a sandbox. With runsc: checkpoint to disk. Fallback: SIGSTOP freezer.
func (m *Manager) Suspend(sb *Sandbox) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.State != StateRunning {
		return errors.New("machine is not running")
	}
	// prefer runsc checkpoint (memory+processes to disk)
	ckpt := filepath.Join(sb.Dir, "checkpoint")
	os.RemoveAll(ckpt)
	os.MkdirAll(ckpt, 0o755)
	args := []string{"--root", m.runscRoot(sb.Runtime), "checkpoint", "--image-path", ckpt}
	if sb.Runtime == RuntimeRunsc && sb.Network == "host" {
		args = append([]string{"--network=host"}, args...)
	}
	args = append(args, sb.MachineID)
	cmd := exec.Command(sb.Runtime, args...)
	if out, err := cmd.CombinedOutput(); err == nil {
		// checkpoint ok; now kill the sandbox process (state is on disk)
		m.destroyLocked(sb)
		sb.State = StateSuspended
		sb.PausedWithCheckpoint = true
		m.save(sb)
		return nil
	} else {
		log.Printf("checkpoint %s failed (%v); using freezer fallback: %s", sb.MachineID, err, out)
	}
	// freezer fallback: SIGSTOP the init process
	pid := m.initPID(sb)
	if pid <= 0 {
		return errors.New("cannot find sandbox process")
	}
	if err := syscall.Kill(pid, syscall.SIGSTOP); err != nil {
		return err
	}
	sb.State = StatePaused
	sb.PausedWithCheckpoint = false
	m.save(sb)
	return nil
}

// Resume wakes a suspended/paused sandbox.
func (m *Manager) Resume(sb *Sandbox, spec *StartSpec) (bool, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.State == StateRunning {
		return true, nil
	}
	if sb.State == StatePaused && !sb.PausedWithCheckpoint {
		pid := m.initPID(sb)
		if pid > 0 {
			if err := syscall.Kill(pid, syscall.SIGCONT); err == nil {
				sb.State = StateRunning
				m.save(sb)
				return true, nil
			}
		}
		// process gone → fresh start
	}
	if sb.PausedWithCheckpoint {
		ckpt := filepath.Join(sb.Dir, "checkpoint")
		if _, err := os.Stat(filepath.Join(ckpt, "checkpoint.img")); err == nil {
			if err := m.restoreLocked(sb, ckpt); err == nil {
				return true, nil
			} else {
				log.Printf("restore %s failed: %v (fresh start)", sb.MachineID, err)
			}
		}
	}
	// fresh start with files
	sb.mu.Unlock()
	err := m.Start(sb, spec)
	sb.mu.Lock()
	return false, err
}

// Stop shuts the sandbox down gracefully, then kills it.
func (m *Manager) Stop(sb *Sandbox) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if sb.State != StateRunning && sb.State != StatePaused {
		return nil
	}
	if sb.State == StatePaused && !sb.PausedWithCheckpoint {
		pid := m.initPID(sb)
		if pid > 0 {
			syscall.Kill(pid, syscall.SIGCONT)
		}
	}
	m.destroyLocked(sb)
	sb.State = StateStopped
	sb.PausedWithCheckpoint = false
	m.save(sb)
	return nil
}

// Restart power-cycles a sandbox.
func (m *Manager) Restart(sb *Sandbox, spec *StartSpec) error {
	_ = m.Stop(sb)
	return m.Start(sb, spec)
}

// Delete removes all on-disk state.
func (m *Manager) Delete(sb *Sandbox) error {
	m.Stop(sb)
	sb.mu.Lock()
	defer sb.mu.Unlock()
	for port := range sb.forwards {
		m.unforwardLocked(sb, port)
	}
	os.RemoveAll(sb.Dir)
	m.mu.Lock()
	delete(m.sandboxes, sb.MachineID)
	m.mu.Unlock()
	return nil
}

func (m *Manager) destroyLocked(sb *Sandbox) {
	if sb.Runtime == "" {
		sb.Runtime = m.Runtime
	}
	// runsc has no "destroy" subcommand — it is "delete" (-force kills a
	// running sandbox). A wrong command here silently leaves the sandbox
	// process alive with its rootfs already replaced on restore.
	args := []string{"--root", m.runscRoot(sb.Runtime), "delete", "-force", sb.MachineID}
	out, err := exec.Command(sb.Runtime, args...).CombinedOutput()
	if err != nil {
		log.Printf("destroy %s: %v: %s", sb.MachineID, err, strings.TrimSpace(string(out)))
	}
}

func (m *Manager) initPID(sb *Sandbox) int {
	args := []string{"--root", m.runscRoot(sb.Runtime), "list", "-format=json"}
	out, err := exec.Command(sb.Runtime, args...).Output()
	if err != nil {
		return -1
	}
	var items []struct {
		ID  string `json:"id"`
		Pid int    `json:"pid"`
	}
	if json.Unmarshal(out, &items) != nil {
		return -1
	}
	for _, it := range items {
		if it.ID == sb.MachineID && it.Pid > 0 {
			return it.Pid
		}
	}
	return -1
}

// CgroupDir returns the cgroup path for a sandbox (best effort).
func (m *Manager) CgroupDir(sb *Sandbox) string {
	return filepath.Join(m.CgroupRoot, sb.MachineID)
}

// ensure cgroup parent exists
func (m *Manager) EnsureCgroup(sb *Sandbox) {
	p := m.CgroupDir(sb)
	os.MkdirAll(p, 0o755)
}

// Status returns a machine-facing status.
func (sb *Sandbox) Status() model.Runtime {
	return model.Runtime{
		Ready:              sb.State == StateRunning,
		ProcessesPreserved: sb.State == StateRunning || sb.State == StateSuspended || sb.State == StatePaused,
	}
}

var _ = bytes.MinRead
var _ = context.Background
