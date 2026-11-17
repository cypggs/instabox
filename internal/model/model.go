// Package model defines the API-facing types shared by control plane and agent.
// Field naming mirrors the InstaBox API where applicable.
package model

import "time"

// ---- Machine ----

type Machine struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DesiredState string `json:"desiredState"`
	State       string `json:"state"`
	Revision    int    `json:"revision"`
	OperationID string `json:"operationId,omitempty"`
	NodeID      string `json:"nodeId,omitempty"`

	Template  *TemplateRef `json:"template,omitempty"`
	Snapshot  *SnapshotRef `json:"snapshot,omitempty"`
	Image     *SnapshotRef `json:"image,omitempty"` // deprecated alias
	Spec      Spec         `json:"spec"`
	Storage   Storage      `json:"storage"`
	Runtime   Runtime      `json:"runtime"`

	AutoSuspendAfterSeconds int    `json:"autoSuspendAfterSeconds"`
	MaxRunMinutes           int    `json:"maxRunMinutes"`
	RunStartedAt            string `json:"runStartedAt,omitempty"`
	ReadyAt                 string `json:"readyAt,omitempty"`
	CooldownUntil           string `json:"cooldownUntil,omitempty"`
	AutoSuspendedAt         string `json:"autoSuspendedAt,omitempty"`
	ArchiveAt               string `json:"archiveAt,omitempty"`
	DeleteAt                string `json:"deleteAt,omitempty"`

	Metadata  map[string]string `json:"metadata"`
	Vault     bool              `json:"vault"`
	GuestUser string            `json:"guestUser"`
	OrgID     string            `json:"orgId"`
	Free      bool              `json:"free"`
	OOMKills  int               `json:"oomKills"`

	EnvNames []string `json:"envNames,omitempty"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type TemplateRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
	Kernel  string `json:"kernel,omitempty"`
}

type SnapshotRef struct {
	ID string `json:"id"`
}

type Spec struct {
	VCPUs             int `json:"vcpus"`
	MemMiB            int `json:"memMiB"`
	PersistentDiskGiB int `json:"persistentDiskGiB"`
	RootDiskGiB       int `json:"rootDiskGiB,omitempty"`
}

type Storage struct {
	SizeGiB     int    `json:"sizeGiB"`
	MountPath   string `json:"mountPath"`
	Persistent  bool   `json:"persistent"`
	Synced      bool   `json:"synced"`
	SnapshotSeq int    `json:"snapshotSeq"`
}

type Runtime struct {
	Ready              bool `json:"ready"`
	ProcessesPreserved bool `json:"processesPreserved"`
}

// ---- Operation ----

type Operation struct {
	ID          string          `json:"id"`
	MachineID   string          `json:"machineId"`
	Kind        string          `json:"kind"`
	Status      string          `json:"status"` // pending|running|succeeded|failed
	Error       *Error          `json:"error,omitempty"`
	Result      any             `json:"result,omitempty"`
	StartedAt   string          `json:"startedAt,omitempty"`
	FinishedAt  string          `json:"finishedAt,omitempty"`
	CreatedAt   string          `json:"createdAt"`
}

// ---- Snapshot ----

type Snapshot struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	OrgID             string       `json:"orgId"`
	SourceMachineID   string       `json:"sourceMachineId"`
	SourceSnapshotSeq int          `json:"sourceSnapshotSeq"`
	Template          TemplateRef  `json:"template"`
	RootSizeBytes     int64        `json:"rootSizeBytes"`
	DataSizeBytes     int64        `json:"dataSizeBytes"`
	DefaultSpec       Spec         `json:"defaultSpec"`
	Status            string       `json:"status"` // saving|ready|error
	RootDigest        string       `json:"rootDigest,omitempty"`
	DataDigest        string       `json:"dataDigest,omitempty"`
	RootOnly          bool         `json:"rootOnly"`
	GuestUser         string       `json:"guestUser"`
	CreatedAt         string       `json:"createdAt"`
}

// ---- Template ----

type Template struct {
	ID          string   `json:"id"`
	Version     string   `json:"version"`
	Digest      string   `json:"digest"`
	Description string   `json:"description"`
	Arch        string   `json:"arch"`
	DefaultSpec Spec     `json:"defaultSpec"`
	Capabilities []string `json:"capabilities"`
	Hidden      bool     `json:"hidden"`
	Creatable   bool     `json:"creatable"`
	Latest      bool     `json:"latest"`
}

// ---- Port ----

type Port struct {
	ID        string `json:"id"`
	MachineID string `json:"machineId"`
	Port      int    `json:"port"`
	Name      string `json:"name"`
	Protocol  string `json:"protocol"` // http|tcp
	URL       string `json:"url,omitempty"`
	TokenURL  string `json:"tokenUrl,omitempty"`
	Access    string `json:"access"` // public|private
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// ---- Usage ----

type Usage struct {
	ComputedAt    string      `json:"computedAt"`
	LogicalBytes  int64       `json:"logicalBytes"`
	StoredBytes   int64       `json:"storedBytes"`
	Machines      int         `json:"machines"`
	Snapshots     int         `json:"snapshots"`
	Limits        UsageLimits `json:"limits"`
	AllocatedDiskGiB int      `json:"allocatedDiskGiB"`
	Cost          Cost        `json:"cost"`
}

type UsageLimits struct {
	MaxMachines   int  `json:"maxMachines"`
	MaxSnapshots  int  `json:"maxSnapshots"`
	MaxStoredGiB  int  `json:"maxStoredGiB"`
	MachineSpecs  []Spec `json:"machineSpecs"`
	LiveSnapshots bool `json:"liveSnapshots"`
}

type Cost struct {
	PeriodStart      string  `json:"periodStart"`
	PeriodEnd        string  `json:"periodEnd"`
	Machines         int     `json:"machines"`
	FreeMachines     int     `json:"freeMachines"`
	FreeSandboxHours float64 `json:"freeSandboxHours"`
	VCPUHours        float64 `json:"vcpuHours"`
	MemGiBHours      float64 `json:"memGiBHours"`
	CostUSD          float64 `json:"costUSD"`
}

// ---- Exec ----

type ExecRequest struct {
	Cmd            string            `json:"cmd"`
	Cmdv           []string          `json:"cmdv,omitempty"` // exact argv, no shell
	Cwd            string            `json:"cwd,omitempty"`
	Envs           map[string]string `json:"envs,omitempty"`
	TimeoutSeconds int               `json:"timeoutSeconds,omitempty"` // default 60, max 600
	Stdin          string            `json:"stdin,omitempty"`
	Background     bool              `json:"background,omitempty"`
	Stream         bool              `json:"stream,omitempty"` // request SSE from API
}

type ExecResult struct {
	ExitCode  int    `json:"exitCode"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Truncated bool   `json:"truncated"`
	JobID     string `json:"jobId,omitempty"`
}

// ---- Files ----

type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	IsLink  bool   `json:"isLink"`
	LinkTo  string `json:"linkTo,omitempty"`
}

// ---- Metrics ----

type MetricSample struct {
	At            time.Time `json:"at"`
	CPUPercent    float64   `json:"cpuPercent"`
	MemUsedBytes  int64     `json:"memUsedBytes"`
	MemTotalBytes int64     `json:"memTotalBytes"`
	NetRxBytes    int64     `json:"netRxBytes"` // delta since previous sample
	NetTxBytes    int64     `json:"netTxBytes"`
	DiskRBytes    int64     `json:"diskReadBytes"`
	DiskWBytes    int64     `json:"diskWrittenBytes"`
}

// ---- Errors ----

type Error struct {
	Code             string `json:"code"`
	Message          string `json:"message"`
	RequestID        string `json:"requestId,omitempty"`
	Retryable        bool   `json:"retryable"`
	OperationID      string `json:"operationId,omitempty"`
	Reason           string `json:"reason,omitempty"`
	Hint             string `json:"hint,omitempty"`
	DocsURL          string `json:"docsUrl,omitempty"`
	RetryAfterSeconds int   `json:"retryAfterSeconds,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// ---- User / orgs ----

type User struct {
	UserID    int    `json:"userId"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Admin     bool   `json:"admin"`
	CSRFToken string `json:"csrfToken,omitempty"`

	AllowedSpecs []Spec `json:"allowedSpecs"`
	Org          Org    `json:"org"`

	IdleChoicesSeconds []int `json:"idleChoicesSeconds"`
	Plans              []Plan `json:"plans"`
}

type Org struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Kind        string `json:"kind"`
	Personal    bool   `json:"personal"`
	Role        string `json:"role,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	AllowedSpecs []Spec `json:"allowedSpecs,omitempty"`
}

type Plan struct {
	Tier          string `json:"tier"`
	MonthlyFeeUSD int    `json:"monthlyFeeUSD"`
	MaxMachines   int    `json:"maxMachines"`
	MaxRunning    int    `json:"maxRunning"`
	MaxVCPUs      int    `json:"maxVCPUs"`
	MaxDiskGiB    int    `json:"maxDiskGiB"`
}

// Machine states
const (
	StateCreating  = "creating"
	StateStarting  = "starting"
	StateRunning   = "running"
	StateStopping  = "stopping"
	StateStopped   = "stopped"
	StateSuspending = "suspending"
	StateSuspended = "suspended"
	StateResuming  = "resuming"
	StateDeleting  = "deleting"
	StateError     = "error"
)

// Desired states
const (
	DesiredRunning   = "running"
	DesiredStopped   = "stopped"
	DesiredSuspended = "suspended"
	DesiredDeleted   = "deleted"
)
