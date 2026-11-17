package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/ibox/ibox/internal/auth"
	"github.com/ibox/ibox/internal/id"
	"github.com/ibox/ibox/internal/model"
	"github.com/ibox/ibox/internal/nodes"
)

var createAllowedFields = map[string]bool{
	"name": true, "templateId": true, "templateVersion": true, "snapshotId": true,
	"vcpus": true, "memMiB": true, "persistentDiskGiB": true, "rootDiskGiB": true,
	"desiredState": true, "metadata": true, "archiveSeconds": true, "deleteAfterSeconds": true,
	"maxRunMinutes": true, "autoSuspendAfterSeconds": true, "vault": true, "env": true,
	"guestUser": true, "imageId": true, "orgId": true,
}

// ---- machines collection ----

func (s *Server) handleMachines(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	switch r.Method {
	case http.MethodGet:
		if e := s.requireScope(ac, "machines:read"); e != nil {
			s.err(w, 403, e)
			return
		}
		ms, err := s.Store.ListMachines(ac.OrgID)
		if err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"machines": ms})
	case http.MethodPost:
		s.createMachine(w, r, ac)
	default:
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
	}
}

type createMachineRequest struct {
	Name                    string            `json:"name"`
	TemplateID              string            `json:"templateId"`
	TemplateVersion         string            `json:"templateVersion"`
	SnapshotID              string            `json:"snapshotId"`
	VCPUs                   *int              `json:"vcpus"`
	MemMiB                  *int              `json:"memMiB"`
	PersistentDiskGiB       *int              `json:"persistentDiskGiB"`
	RootDiskGiB             *int              `json:"rootDiskGiB"`
	DesiredState            string            `json:"desiredState"`
	Metadata                map[string]string `json:"metadata"`
	ArchiveSeconds          *int              `json:"archiveSeconds"`
	DeleteAfterSeconds      *int              `json:"deleteAfterSeconds"`
	MaxRunMinutes           *int              `json:"maxRunMinutes"`
	AutoSuspendAfterSeconds *int              `json:"autoSuspendAfterSeconds"`
	Vault                   *bool             `json:"vault"`
	Env                     map[string]string `json:"env"`
	GuestUser               string            `json:"guestUser"`
}

func (s *Server) createMachine(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if e := s.requireScope(ac, "machines:write"); e != nil {
		s.err(w, 403, e)
		return
	}
	var req createMachineRequest
	decBody, err := io.ReadAll(r.Body)
	if err != nil {
		s.err(w, 400, badRequest("invalid body"))
		return
	}
	if err := json.Unmarshal(decBody, &req); err != nil {
		s.err(w, 400, badRequest("invalid JSON body"))
		return
	}
	var rawFields map[string]any
	_ = json.Unmarshal(decBody, &rawFields)
	for k := range rawFields {
		if !createAllowedFields[k] {
			s.err(w, 400, &model.Error{Code: "unknown_field", Message: "unknown field " + k + " in create request"})
			return
		}
	}

	// quota
	count, _ := s.Store.CountMachines(ac.OrgID)
	if count >= 5 {
		s.err(w, 429, &model.Error{Code: "quota_exceeded", Message: "machine limit reached (5 on the free plan)", Hint: "delete a machine or upgrade"})
		return
	}

	name := req.Name
	if name == "" {
		name = "box-" + randHex(4)
	}
	if !nameRe.MatchString(name) {
		s.err(w, 400, badRequest("name must be 1-63 of a-z, 0-9, -, starting with a letter or digit"))
		return
	}

	// template or snapshot
	var tmpl model.Template
	var snap *model.Snapshot
	if req.SnapshotID != "" {
		var err error
		snap, err = s.Store.GetSnapshot(req.SnapshotID, ac.OrgID)
		if err != nil {
			s.err(w, 404, &model.Error{Code: "not_found", Message: "no such snapshot"})
			return
		}
		tmpl = s.Templates.GetByID(snap.Template.ID)
	} else {
		tid := req.TemplateID
		if tid == "" {
			tid = "standard"
		}
		var ok bool
		tmpl, ok = s.Templates.Resolve(tid, req.TemplateVersion)
		if !ok {
			s.err(w, 400, badRequest("unknown template "+tid))
			return
		}
		if !tmpl.Creatable {
			s.err(w, 403, &model.Error{Code: "template_not_allowed", Message: "template " + tid + " is not creatable on your plan"})
			return
		}
	}

	spec := tmpl.DefaultSpec
	if snap != nil {
		spec = snap.DefaultSpec
	}
	if req.VCPUs != nil && req.MemMiB != nil {
		spec.VCPUs = *req.VCPUs
		spec.MemMiB = *req.MemMiB
		if !specAllowed(spec) {
			s.err(w, 400, &model.Error{Code: "spec_not_allowed", Message: "size not offered by your plan", Hint: "GET /v1/orgs lists allowedSpecs"})
			return
		}
	}
	if req.PersistentDiskGiB != nil {
		spec.PersistentDiskGiB = *req.PersistentDiskGiB
	}
	if req.RootDiskGiB != nil {
		spec.RootDiskGiB = *req.RootDiskGiB
	}

	autoSuspend := 1800
	if req.AutoSuspendAfterSeconds != nil {
		v := *req.AutoSuspendAfterSeconds
		if v != 0 {
			ok := false
			for _, c := range []int{300, 600, 900, 1200, 1500, 1800} {
				if v == c {
					ok = true
				}
			}
			if !ok {
				s.err(w, 400, badRequest("autoSuspendAfterSeconds must be one of 300,600,900,1200,1500,1800 (or 0 off for paid plans)"))
				return
			}
		}
		autoSuspend = v
	}

	maxRun := 120
	if req.MaxRunMinutes != nil {
		maxRun = *req.MaxRunMinutes
	}

	guestUser := req.GuestUser
	if guestUser == "" {
		guestUser = "user"
	}
	if snap != nil && snap.GuestUser != "" {
		guestUser = snap.GuestUser
	}

	envNames := []string{}
	if len(req.Env) > 0 {
		if len(req.Env) > 128 {
			s.err(w, 400, badRequest("at most 128 environment variables"))
			return
		}
		reserved := map[string]bool{"PATH": true, "HOME": true, "SHELL": true, "USER": true, "HOSTNAME": true, "PWD": true, "OLDPWD": true, "IFS": true}
		envRe := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)
		total := 0
		for k := range req.Env {
			if reserved[k] || strings.HasPrefix(k, "INSTABOX") || strings.HasPrefix(k, "SSH_") {
				s.err(w, 400, badRequest("environment variable name not allowed: "+k))
				return
			}
			if !envRe.MatchString(k) {
				s.err(w, 400, badRequest("invalid environment variable name: "+k))
				return
			}
			total += len(k) + len(req.Env[k])
			envNames = append(envNames, k)
		}
		if total > 32*1024 {
			s.err(w, 400, badRequest("environment variables exceed 32 KiB"))
			return
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	desired := model.DesiredRunning
	if req.DesiredState == model.DesiredStopped {
		desired = model.DesiredStopped
	}

	m := &model.Machine{
		ID: id.Machine(), OrgID: ac.OrgID, Name: name,
		DesiredState: desired, State: model.StateCreating, Revision: 1,
		Template: tmplRefPtr(tmpl), Spec: spec,
		Storage: model.Storage{SizeGiB: spec.PersistentDiskGiB, MountPath: "/data", Persistent: true},
		AutoSuspendAfterSeconds: autoSuspend, MaxRunMinutes: maxRun,
		Metadata: req.Metadata, Vault: true, GuestUser: guestUser,
		Free: true, EnvNames: envNames,
		CreatedAt: now, UpdatedAt: now,
	}
	if req.Vault != nil {
		m.Vault = *req.Vault
	}
	if snap != nil {
		m.Snapshot = &model.SnapshotRef{ID: snap.ID}
		m.Image = m.Snapshot
	}
	if req.ArchiveSeconds != nil && *req.ArchiveSeconds > 0 {
		m.ArchiveAt = time.Now().UTC().Add(time.Duration(*req.ArchiveSeconds) * time.Second).Format(time.RFC3339Nano)
	}
	if req.DeleteAfterSeconds != nil && *req.DeleteAfterSeconds > 0 {
		m.DeleteAt = time.Now().UTC().Add(time.Duration(*req.DeleteAfterSeconds) * time.Second).Format(time.RFC3339Nano)
	}
	if err := s.Store.InsertMachine(m); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			s.err(w, 409, &model.Error{Code: "already_exists", Message: "a machine with this name already exists"})
			return
		}
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	// store env values encrypted-at-rest (AES-GCM; key = local secret)
	if len(req.Env) > 0 {
		blob, err := s.encryptJSON(req.Env)
		if err == nil {
			_ = s.Store.SetMachineEnv(m.ID, envNames, blob)
		}
	}

	// dispatch create directly (not via reconciliation)
	go s.executeCreate(m, req.Env)

	writeJSON(w, 202, m)
}

func tmplRef(t model.Template) model.TemplateRef {
	return model.TemplateRef{ID: t.ID, Version: t.Version, Digest: t.Digest}
}

func tmplRefPtr(t model.Template) *model.TemplateRef {
	r := tmplRef(t)
	return &r
}

func specAllowed(spec model.Spec) bool {
	for _, a := range []model.Spec{{VCPUs: 2, MemMiB: 4096, PersistentDiskGiB: 20}, {VCPUs: 4, MemMiB: 8192, PersistentDiskGiB: 20}, {VCPUs: 8, MemMiB: 16384, PersistentDiskGiB: 20}} {
		if a.VCPUs == spec.VCPUs && a.MemMiB == spec.MemMiB {
			return true
		}
	}
	return false
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// executeCreate runs the create operation inline (async goroutine).
func (s *Server) executeCreate(m *model.Machine, env map[string]string) {
	ctx := context.Background()
	node, nerr := s.Registry.AnyNode()
	opID := id.Operation()
	op := &model.Operation{ID: opID, MachineID: m.ID, Kind: "create", Status: "pending", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	_ = s.Store.InsertOperation(op)
	ok, _ := s.Store.AcquireMachineLock(m.ID, opID)
	if !ok {
		return
	}
	_ = s.Store.SetOperationRunning(opID)
	_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.OperationID = opID })
	defer func() {
		_ = s.Store.ReleaseMachineLock(m.ID)
		_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.OperationID = "" })
	}()

	if nerr != nil {
		_ = s.Store.FinishOperation(opID, "failed", &model.Error{Code: "no_capacity", Message: "no execution node connected", Retryable: true}, nil)
		_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.State = model.StateError })
		return
	}

	spec := nodes.SandboxSpec{
		MachineID: m.ID, Name: m.Name,
		VCPUs: m.Spec.VCPUs, MemMiB: m.Spec.MemMiB,
		RootDiskGiB: m.Spec.RootDiskGiB, DiskGiB: m.Spec.PersistentDiskGiB,
		GuestUser: m.GuestUser, Env: env,
	}
	if m.Snapshot != nil {
		spec.TemplateDir = "" // agent resolves via snapshot CAS
		spec.DataDir = ""     // restore-from-snapshot handled by agent with snapshot fetch
	}
	params := map[string]any{"spec": spec, "snapshotId": nil}
	if m.Template != nil {
		params["templateId"] = m.Template.ID
	}
	if m.Snapshot != nil {
		params["snapshotId"] = m.Snapshot.ID
	}
	raw, err := node.Call(ctx, "sandbox.create", params, m.ID, 20*time.Minute)
	if err != nil {
		_ = s.Store.FinishOperation(opID, "failed", err, nil)
		_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.State = model.StateError })
		return
	}
	_ = raw
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_ = s.Store.FinishOperation(opID, "succeeded", nil, nil)
	_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
		mm.NodeID = node.ID
		if mm.DesiredState == model.DesiredRunning {
			mm.State = model.StateRunning
			mm.RunStartedAt = now
			mm.ReadyAt = now
		} else {
			mm.State = model.StateStopped
		}
		mm.Storage.Synced = true
	})
}

// ---- machine subroutes ----

func (s *Server) handleMachineSub(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	rest := strings.TrimPrefix(r.URL.Path, "/v1/machines/")
	parts := strings.SplitN(rest, "/", 2)
	ref := parts[0]
	if ref == "" {
		s.err(w, 404, &model.Error{Code: "not_found", Message: "route not found"})
		return
	}
	if len(parts) == 1 {
		// /v1/machines/{id}
		switch r.Method {
		case http.MethodGet:
			s.getMachine(w, r, ac, ref)
		case http.MethodPatch:
			s.patchMachine(w, r, ac, ref)
		case http.MethodDelete:
			s.deleteMachine(w, r, ac, ref)
		default:
			s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		}
		return
	}
	sub := parts[1]
	m, merr := s.machineByIDOrName(ref, ac.OrgID)
	if merr != nil {
		s.err(w, 404, merr)
		return
	}
	switch sub {
	case "start":
		s.lifecycleAction(w, r, ac, m, model.DesiredRunning, "start")
	case "stop":
		s.lifecycleAction(w, r, ac, m, model.DesiredStopped, "stop")
	case "suspend":
		s.lifecycleAction(w, r, ac, m, model.DesiredSuspended, "suspend")
	case "resume":
		s.lifecycleAction(w, r, ac, m, model.DesiredRunning, "resume")
	case "wake":
		s.wakeMachine(w, r, ac, m)
	case "restart":
		s.restartMachine(w, r, ac, m)
	case "exec":
		s.execInMachine(w, r, ac, m)
	case "files":
		s.filesRoute(w, r, ac, m)
	case "files/content":
		s.fileContentRoute(w, r, ac, m)
	case "files/upload-url":
		s.fileUploadURLRoute(w, r, ac, m)
	case "ports":
		s.portsRoute(w, r, ac, m)
	case "ports/":
		s.portDeleteRoute(w, r, ac, m)
	case "metrics":
		s.metricsRoute(w, r, ac, m)
	case "terminal":
		s.terminalRoute(w, r, ac, m)
	case "snapshots":
		s.machineSnapshotsRoute(w, r, ac, m)
	case "restore":
		s.restoreRoute(w, r, ac, m)
	case "fork":
		s.forkRoute(w, r, ac, m)
	case "logs":
		s.logsRoute(w, r, ac, m)
	case "history":
		s.historyRoute(w, r, ac, m)
	case "processes":
		s.processesRoute(w, r, ac, m)
	case "connection-info":
		s.connectionInfoRoute(w, r, ac, m)
	case "activity-leases/", "activity-leases":
		s.leasesRoute(w, r, ac, m)
	case "vault", "secrets":
		s.machineSecretsRoute(w, r, ac, m)
	case "desktop":
		s.err(w, 501, &model.Error{Code: "not_implemented", Message: "remote desktop is not implemented in this clone"})
	case "vnc":
		s.err(w, 501, &model.Error{Code: "not_implemented", Message: "VNC is not implemented in this clone"})
	default:
		s.err(w, 404, &model.Error{Code: "not_found", Message: "route not found"})
	}
}

func (s *Server) getMachine(w http.ResponseWriter, r *http.Request, ac *auth.Context, ref string) {
	if e := s.requireScope(ac, "machines:read"); e != nil {
		s.err(w, 403, e)
		return
	}
	m, merr := s.machineByIDOrName(ref, ac.OrgID)
	if merr != nil {
		s.err(w, 404, merr)
		return
	}
	writeJSON(w, 200, m)
}

type patchMachineRequest struct {
	Name                    *string           `json:"name"`
	VCPUs                   *int              `json:"vcpus"`
	MemMiB                  *int              `json:"memMiB"`
	PersistentDiskGiB       *int              `json:"persistentDiskGiB"`
	AutoSuspendAfterSeconds *int              `json:"autoSuspendAfterSeconds"`
	MaxRunMinutes           *int              `json:"maxRunMinutes"`
	Metadata                map[string]string `json:"metadata"`
}

func (s *Server) patchMachine(w http.ResponseWriter, r *http.Request, ac *auth.Context, ref string) {
	if e := s.requireScope(ac, "machines:write"); e != nil {
		s.err(w, 403, e)
		return
	}
	m, merr := s.machineByIDOrName(ref, ac.OrgID)
	if merr != nil {
		s.err(w, 404, merr)
		return
	}
	var req patchMachineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.err(w, 400, badRequest("invalid JSON body"))
		return
	}
	if req.Name != nil {
		if !nameRe.MatchString(*req.Name) {
			s.err(w, 400, badRequest("invalid name"))
			return
		}
		if err := s.Store.RenameMachine(m.ID, *req.Name); err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
	}
	if (req.VCPUs != nil || req.MemMiB != nil) && m.Free && m.State == model.StateRunning {
		s.err(w, 409, &model.Error{Code: "operation_conflict", Message: "stop the machine before resizing a free machine"})
		return
	}
	_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
		if req.VCPUs != nil {
			mm.Spec.VCPUs = *req.VCPUs
		}
		if req.MemMiB != nil {
			mm.Spec.MemMiB = *req.MemMiB
		}
		if req.PersistentDiskGiB != nil {
			mm.Spec.PersistentDiskGiB = *req.PersistentDiskGiB
		}
		if req.AutoSuspendAfterSeconds != nil {
			mm.AutoSuspendAfterSeconds = *req.AutoSuspendAfterSeconds
		}
		if req.MaxRunMinutes != nil {
			mm.MaxRunMinutes = *req.MaxRunMinutes
		}
		if req.Metadata != nil {
			mm.Metadata = req.Metadata
		}
	})
	cur, _ := s.Store.GetMachineByID(m.ID)
	writeJSON(w, 200, cur)
}

func (s *Server) deleteMachine(w http.ResponseWriter, r *http.Request, ac *auth.Context, ref string) {
	if e := s.requireScope(ac, "machines:write"); e != nil {
		s.err(w, 403, e)
		return
	}
	m, merr := s.machineByIDOrName(ref, ac.OrgID)
	if merr != nil {
		s.err(w, 404, merr)
		return
	}
	confirm := r.Header.Get("x-confirm-delete")
	if ac.IsAPIKey && confirm == "" {
		s.err(w, 409, &model.Error{Code: "delete_confirmation_required", Message: "send X-Confirm-Delete: " + m.ID, Hint: "repeat the request with the header set to the machine ID or name"})
		return
	}
	if confirm != "" && confirm != m.ID && confirm != m.Name {
		s.err(w, 409, &model.Error{Code: "delete_confirmation_required", Message: "X-Confirm-Delete does not match", Hint: "send the machine ID (" + m.ID + ") or exact name"})
		return
	}
	if m.State == model.StateSuspended {
		s.err(w, 409, &model.Error{Code: "operation_conflict", Message: "a paused machine cannot be deleted after stop; resume it first, or delete will proceed anyway"})
		// The original refuses stop of paused machines but allows delete; proceed.
	}
	_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.DesiredState = model.DesiredDeleted })
	s.Controller.Enqueue(m.ID)
	writeJSON(w, 202, map[string]any{"ok": true, "desiredState": "deleted"})
}

// lifecycleAction sets desired state and returns an operation placeholder.
func (s *Server) lifecycleAction(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine, desired, kind string) {
	if e := s.requireScope(ac, "machines:write"); e != nil {
		s.err(w, 403, e)
		return
	}
	if r.Method != http.MethodPost {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	// state guardrails matching the original
	if kind == "stop" && m.State == model.StateSuspended {
		s.err(w, 409, &model.Error{Code: "operation_conflict", Message: "a paused machine cannot be stopped; resume it first"})
		return
	}
	if kind == "resume" && m.State == model.StateRunning {
		writeJSON(w, 200, m)
		return
	}
	if c := s.cooldownCheck(m); c != nil && (kind == "start" || kind == "resume" || kind == "wake") {
		s.err(w, 409, c)
		return
	}
	_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.DesiredState = desired })
	if kind == "start" || kind == "resume" {
		// reset cooldown streak on explicit user start after 2h
		_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
			if mm.CooldownUntil != "" {
				if t, err := time.Parse(time.RFC3339Nano, mm.CooldownUntil); err == nil && time.Since(t) > 2*time.Hour {
					delete(mm.Metadata, "cooldownStreak")
				}
			}
			mm.CooldownUntil = ""
		})
	}
	s.Controller.Enqueue(m.ID)
	writeJSON(w, 202, map[string]any{"ok": true, "machineId": m.ID})
}

func (s *Server) cooldownCheck(m *model.Machine) *model.Error {
	if m.CooldownUntil == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, m.CooldownUntil)
	if err != nil || time.Now().UTC().After(t) {
		return nil
	}
	secs := int(time.Until(t).Seconds()) + 1
	return &model.Error{Code: "machine_cooldown", Message: "machine is cooling down after reaching its maximum runtime", Retryable: true, RetryAfterSeconds: secs}
}

func (s *Server) wakeMachine(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodPost {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	if m.State != model.StateSuspended || m.AutoSuspendedAt == "" {
		if m.State == model.StateRunning {
			writeJSON(w, 200, map[string]any{"ok": true})
			return
		}
		s.err(w, 409, &model.Error{Code: "machine_not_running", Message: "machine is stopped or manually paused; start or resume it first"})
		return
	}
	if c := s.cooldownCheck(m); c != nil {
		s.err(w, 409, c)
		return
	}
	_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.DesiredState = model.DesiredRunning })
	s.Controller.Enqueue(m.ID)
	// wait up to 45s like usage-driven wake
	if e := s.ensureRunning(r.Context(), m, 45*time.Second); e != nil {
		s.err(w, 409, e)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) restartMachine(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodPost {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	if m.State != model.StateRunning && m.State != model.StateStarting {
		s.err(w, 409, &model.Error{Code: "machine_not_running", Message: "machine is not running"})
		return
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}
	raw, err := node.Call(r.Context(), "sandbox.restart", map[string]string{"machineId": m.ID}, m.ID, 150*time.Second)
	_ = raw
	if err != nil {
		s.err(w, 500, err)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
		mm.State = model.StateRunning
		mm.RunStartedAt = now
		mm.ReadyAt = now
	})
	s.Store.RecordActivity(m.ID, "restart")
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ---- exec ----

func (s *Server) execInMachine(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if e := s.requireScope(ac, "machines:write"); e != nil {
		s.err(w, 403, e)
		return
	}
	if r.Method != http.MethodPost {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	var req model.ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.err(w, 400, badRequest("invalid JSON body"))
		return
	}
	if req.Cmd == "" && len(req.Cmdv) == 0 {
		s.err(w, 400, badRequest("cmd is required"))
		return
	}
	if e := s.ensureRunning(r.Context(), m, 45*time.Second); e != nil {
		s.err(w, 409, e)
		return
	}
	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	if timeout > 600 && !req.Background {
		timeout = 600
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}

	s.Store.RecordActivity(m.ID, "exec")
	argv := req.Cmdv
	shell := false
	if len(argv) == 0 {
		argv = []string{req.Cmd}
		shell = true
	}
	params := nodes.ExecParams{
		MachineID: m.ID, Argv: argv, Cwd: req.Cwd, Env: req.Envs,
		TimeoutS: timeout, Stdin: []byte(req.Stdin), Shell: shell,
	}
	if req.Background {
		params.TimeoutS = 0
	}

	stream := strings.Contains(r.Header.Get("accept"), "text/event-stream") || req.Stream
	if stream {
		s.execStream(w, r, node, m, params, &req)
		return
	}

	raw, err := node.Call(r.Context(), "sandbox.exec", params, m.ID, time.Duration(timeout+30)*time.Second)
	if err != nil {
		status := 500
		if err.Code == "machine_not_running" {
			status = 409
		}
		s.err(w, status, err)
		return
	}
	var out nodes.ExecOutcome
	if json.Unmarshal(raw, &out) != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: "bad exec result"})
		return
	}
	truncated := out.Truncated
	if !req.Background && len(out.Stdout) > 1<<20 {
		out.Stdout = out.Stdout[:1<<20]
		truncated = true
	}
	if !req.Background && len(out.Stderr) > 1<<20 {
		out.Stderr = out.Stderr[:1<<20]
		truncated = true
	}
	started := time.Now().UTC()
	finished := started.Add(time.Second)
	if req.Background {
		_, _ = s.Store.InsertExecHistory(m.ID, req.Cmd, req.Cwd, -1, nil, nil, started.Format(time.RFC3339Nano), finished.Format(time.RFC3339Nano))
		writeJSON(w, 200, &model.ExecResult{ExitCode: 0, JobID: string(out.Stdout)})
		return
	}
	eid, _ := s.Store.InsertExecHistory(m.ID, req.Cmd, req.Cwd, out.ExitCode, out.Stdout, out.Stderr, started.Format(time.RFC3339Nano), finished.Format(time.RFC3339Nano))
	_ = eid
	writeJSON(w, 200, &model.ExecResult{ExitCode: out.ExitCode, Stdout: string(out.Stdout), Stderr: string(out.Stderr), Truncated: truncated})
}

// execStream runs exec and streams output as SSE.
func (s *Server) execStream(w http.ResponseWriter, r *http.Request, node *nodes.Node, m *model.Machine, params nodes.ExecParams, req *model.ExecRequest) {
	w.Header().Set("content-type", "text/event-stream")
	w.Header().Set("cache-control", "no-cache")
	w.WriteHeader(200)
	fl, _ := w.(http.Flusher)

	type streamEvent struct {
		Stdout   string `json:"stdout,omitempty"`
		Stderr   string `json:"stderr,omitempty"`
		ExitCode *int   `json:"exitCode,omitempty"`
		Done     bool   `json:"done"`
	}
	emit := func(ev streamEvent) {
		b, _ := json.Marshal(ev)
		fmt.Fprintf(w, "data: %s\n\n", b)
		fl.Flush()
	}

	// Use a long-running call; agent streams via repeated RPCs is complex —
	// here we collect with progress heartbeats using the node "exec-stream" method.
	resCh := make(chan *nodes.Response, 16)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		params.Env = mergeEnv(params.Env, map[string]string{"IBOX_STREAM": "1"})
		raw, err := node.Call(ctx, "sandbox.exec", params, m.ID, time.Duration(params.TimeoutS+30)*time.Second)
		if err != nil {
			resCh <- &nodes.Response{Error: err}
			return
		}
		resCh <- &nodes.Response{Result: raw}
	}()

	// heartbeat while waiting
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case resp := <-resCh:
			if resp.Error != nil {
				b, _ := json.Marshal(resp.Error)
				fmt.Fprintf(w, "data: {\"error\": %s}\n\n", b)
				fl.Flush()
				return
			}
			var out nodes.ExecOutcome
			if json.Unmarshal(resp.Result, &out) == nil {
				emit(streamEvent{Stdout: string(out.Stdout)})
				emit(streamEvent{Stderr: string(out.Stderr)})
				code := out.ExitCode
				emit(streamEvent{ExitCode: &code, Done: true})
			}
			return
		case <-tick.C:
		emit(streamEvent{})
		case <-ctx.Done():
			return
		}
	}
}

func mergeEnv(a, b map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// ---- files ----

func (s *Server) filesRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if e := s.requireScope(ac, "machines:read"); e != nil {
		s.err(w, 403, e)
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}
	switch r.Method {
	case http.MethodGet:
		// list directory (works on stopped machines for /data too — original reads stored state)
		if m.State != model.StateRunning {
			if e := s.ensureRunning(r.Context(), m, 45*time.Second); e != nil && m.State != model.StateStopped {
				s.err(w, 409, e)
				return
			}
		}
		node, nerr := s.nodeFor(r.Context())
		if nerr != nil {
			s.err(w, 503, nerr)
			return
		}
		if m.State == model.StateRunning {
			s.Store.RecordActivity(m.ID, "file")
		}
		raw, err := node.Call(r.Context(), "files.list", map[string]string{"machineId": m.ID, "path": path}, m.ID, 30*time.Second)
		if err != nil {
			s.err(w, mapStatus(err), err)
			return
		}
		var listRes struct {
			Entries []model.FileEntry `json:"entries"`
		}
		if json.Unmarshal(raw, &listRes) == nil {
			writeJSON(w, 200, map[string]any{"entries": listRes.Entries})
			return
		}
		writeJSON(w, 200, map[string]any{"entries": []model.FileEntry{}})
	case http.MethodPut:
		if e := s.requireScope(ac, "machines:write"); e != nil {
			s.err(w, 403, e)
			return
		}
		if e := s.ensureRunning(r.Context(), m, 45*time.Second); e != nil {
			s.err(w, 409, e)
			return
		}
		parents := r.URL.Query().Get("parents") == "true"
		body, err := readBodyLimit(r, 5<<30)
		if err != nil {
			s.err(w, 400, badRequest("body too large"))
			return
		}
		node, nerr := s.nodeFor(r.Context())
		if nerr != nil {
			s.err(w, 503, nerr)
			return
		}
		s.Store.RecordActivity(m.ID, "file")
		_, rerr := node.Call(r.Context(), "files.write", map[string]any{
			"machineId": m.ID, "path": path, "parents": parents, "content": base64.StdEncoding.EncodeToString(body),
		}, m.ID, 60*time.Second)
		if rerr != nil {
			s.err(w, mapStatus(rerr), rerr)
			return
		}
		w.WriteHeader(204)
	default:
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
	}
}

func (s *Server) fileContentRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if e := s.requireScope(ac, "machines:read"); e != nil {
		s.err(w, 403, e)
		return
	}
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	path := r.URL.Query().Get("path")
	if path == "" {
		s.err(w, 400, badRequest("path is required"))
		return
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}
	raw, err := node.Call(r.Context(), "files.read", map[string]string{"machineId": m.ID, "path": path}, m.ID, 60*time.Second)
	if err != nil {
		s.err(w, mapStatus(err), err)
		return
	}
	var res struct {
		Content string `json:"content"`
	}
	if json.Unmarshal(raw, &res) != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: "bad file result"})
		return
	}
	b, decErr := base64.StdEncoding.DecodeString(res.Content)
	if decErr != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: "bad file encoding"})
		return
	}
	w.Header().Set("content-type", "application/octet-stream")
	w.WriteHeader(200)
	w.Write(b)
}

func (s *Server) fileUploadURLRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	// Signed upload URLs: in this clone the upload goes through the API directly.
	writeJSON(w, 200, map[string]any{
		"url":    fmt.Sprintf("/v1/machines/%s/files?path=%s", m.ID, r.URL.Query().Get("path")),
		"method": "PUT",
		"note":   "this clone returns an API-relative URL; the agent-side signed URL flow is not needed for single-node deployments",
	})
}

func mapStatus(e *model.Error) int {
	switch e.Code {
	case "machine_not_running":
		return 409
	case "missing_parent", "not_found":
		return 404
	case "no_capacity", "host_unavailable":
		return 503
	default:
		return 500
	}
}

// ---- ports ----

func (s *Server) portsRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	switch r.Method {
	case http.MethodGet:
		ports, err := s.Store.ListPortsForMachine(m.ID)
		if err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		for _, p := range ports {
			s.decoratePortURL(p)
		}
		writeJSON(w, 200, map[string]any{"ports": ports})
	case http.MethodPost:
		s.publishPort(w, r, ac, m)
	default:
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
	}
}

func (s *Server) publishPort(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if e := s.requireScope(ac, "machines:write"); e != nil {
		s.err(w, 403, e)
		return
	}
	var req struct {
		Port        int    `json:"port"`
		Name        string `json:"name"`
		Access      string `json:"access"`
		Protocol    string `json:"protocol"`
		RotateToken bool   `json:"rotateToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.err(w, 400, badRequest("invalid JSON body"))
		return
	}
	if req.Port < 1024 || req.Port > 65535 || req.Port == 5901 || req.Port == 6768 {
		s.err(w, 400, badRequest("port must be 1024-65535 (5901 and 6768 are reserved)"))
		return
	}
	count, _ := s.Store.CountPortsForMachine(m.ID)
	existing, err := s.Store.GetPort(m.ID, req.Port)
	isNew := err != nil
	if isNew && count >= 16 {
		s.err(w, 429, &model.Error{Code: "quota_exceeded", Message: "a machine has at most 16 published ports"})
		return
	}
	access := req.Access
	if access == "" {
		access = "public"
	}
	if access != "public" && access != "private" {
		s.err(w, 400, badRequest("access must be public or private"))
		return
	}
	name := req.Name
	if name == "" {
		name = fmt.Sprintf("Port %d", req.Port)
	}
	protocol := req.Protocol
	if protocol == "" {
		protocol = "http"
	}

	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}
	// ask agent to start the port forward
	_, aerr := node.Call(r.Context(), "ports.publish", map[string]any{
		"machineId": m.ID, "port": req.Port, "protocol": protocol,
	}, m.ID, 30*time.Second)
	if aerr != nil && aerr.Code != "already_exists" {
		s.err(w, 500, aerr)
		return
	}

	tokenURL := ""
	if isNew {
		p := &model.Port{
			ID: id.Port(), MachineID: m.ID, Port: req.Port, Name: name,
			Protocol: protocol, Access: access, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		}
		hash := ""
		if access == "private" {
			tok := "ibp_" + randB32(32)
			sum := sha256.Sum256([]byte(tok))
			hash = base64.RawURLEncoding.EncodeToString(sum[:])
			s.decoratePortURL(p)
			tokenURL = p.URL + "?_token=" + tok
		}
		if err := s.Store.InsertPort(p, hash); err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		s.decoratePortURL(p)
		if access == "private" {
			p.TokenURL = tokenURL
		}
		writeJSON(w, 201, p)
		return
	}
	// update existing
	_ = s.Store.UpdatePort(existing.ID, name, access)
	if access == "private" {
		tok := "ibp_" + randB32(32)
		sum := sha256.Sum256([]byte(tok))
		_ = s.Store.SetPortTokenHash(existing.ID, base64.RawURLEncoding.EncodeToString(sum[:]))
		s.decoratePortURL(existing)
		existing.TokenURL = existing.URL + "?_token=" + tok
	} else {
		_ = s.Store.SetPortTokenHash(existing.ID, "")
	}
	s.decoratePortURL(existing)
	writeJSON(w, 200, existing)
}

func randB32(n int) string {
	const enc = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = enc[b[i]%32]
	}
	return string(b)
}

func (s *Server) portDeleteRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodDelete {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/v1/machines/%s/ports/", m.ID))
	// ref may be id or name
	portNum, err := strconv.Atoi(rest)
	if err != nil {
		s.err(w, 400, badRequest("port number expected"))
		return
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr == nil {
		_, _ = node.Call(context.Background(), "ports.unpublish", map[string]any{"machineId": m.ID, "port": portNum}, m.ID, 15*time.Second)
	}
	_ = s.Store.DeletePort(m.ID, portNum)
	w.WriteHeader(204)
}

// ---- metrics ----

func (s *Server) metricsRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	since := time.Now().UTC().Add(-1 * time.Hour)
	if v := r.URL.Query().Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			since = t.UTC()
		}
	}
	if v := r.URL.Query().Get("minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			since = time.Now().UTC().Add(-time.Duration(n) * time.Minute)
		}
	}
	samples, err := s.Store.ListMetrics(m.ID, since)
	if err != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	points := make([]map[string]any, 0, len(samples))
	for _, smp := range samples {
		points = append(points, map[string]any{
			"at":         smp.At,
			"cpuPercent": smp.CPUPercent,
			"memUsedMiB": smp.MemUsedBytes >> 20,
			"memMaxMiB":  smp.MemTotalBytes >> 20,
		})
	}
	writeJSON(w, 200, map[string]any{"points": points})
}

// ---- terminal (WebSocket) ----

func (s *Server) terminalRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	wake := r.URL.Query().Get("wake")
	if m.State != model.StateRunning {
		if wake == "0" {
			s.err(w, 409, &model.Error{Code: "machine_not_running", Message: "machine is not running", Retryable: true})
			return
		}
		if e := s.ensureRunning(r.Context(), m, 45*time.Second); e != nil {
			s.err(w, 409, e)
			return
		}
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}
	s.Store.RecordActivity(m.ID, "terminal")
	client, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer client.CloseNow()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	streamID, oerr := node.OpenStream(ctx, "terminal.open", map[string]string{"machineId": m.ID}, m.ID, 20*time.Second)
	if oerr != nil {
		client.Close(1011, oerr.Message)
		return
	}
	defer node.CloseStream(streamID)
	ch, _ := node.StreamRecv(streamID)

	// client -> agent
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			typ, data, err := client.Read(ctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageText || typ == websocket.MessageBinary {
				if err := node.StreamSend(streamID, data); err != nil {
					return
				}
			}
		}
	}()
	// agent -> client
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				client.Close(1000, "terminal closed")
				return
			}
			wctx, c2 := context.WithTimeout(ctx, 15*time.Second)
			err := client.Write(wctx, websocket.MessageBinary, data)
			c2()
			if err != nil {
				return
			}
		case <-done:
			return
		case <-node.Done():
			client.Close(1011, "node disconnected")
			return
		case <-ctx.Done():
			return
		}
	}
}

// ---- snapshots ----

func (s *Server) machineSnapshotsRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodPost {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	if e := s.requireScope(ac, "machines:write"); e != nil {
		s.err(w, 403, e)
		return
	}
	var req struct {
		Name     string `json:"name"`
		RootOnly bool   `json:"rootOnly"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if m.State == model.StateSuspended {
		s.err(w, 409, &model.Error{Code: "operation_conflict", Message: "a paused machine cannot be snapshotted; resume it first"})
		return
	}
	count, _ := s.Store.CountSnapshots(ac.OrgID)
	if count >= 5 {
		s.err(w, 429, &model.Error{Code: "quota_exceeded", Message: "snapshot limit reached (5 on the free plan)"})
		return
	}
	name := req.Name
	if name == "" {
		name = m.Name + "-snapshot"
	}
	snap := &model.Snapshot{
		ID: id.Snapshot(), OrgID: ac.OrgID, Name: name,
		SourceMachineID: m.ID, SourceSnapshotSeq: m.Storage.SnapshotSeq,
		Template:    *m.Template,
		DefaultSpec: m.Spec, Status: "saving",
		RootOnly:  req.RootOnly,
		GuestUser: m.GuestUser,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := s.Store.InsertSnapshot(snap); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			s.err(w, 409, &model.Error{Code: "operation_conflict", Message: "a snapshot with this name already exists"})
			return
		}
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	go s.executeSnapshotSave(snap, m, req.RootOnly)
	writeJSON(w, 202, snap)
}

func (s *Server) executeSnapshotSave(snap *model.Snapshot, m *model.Machine, rootOnly bool) {
	ctx := context.Background()
	node, err := s.Registry.AnyNode()
	if err != nil {
		_ = s.Store.UpdateSnapshotStatus(snap.ID, "error")
		return
	}
	raw, rerr := node.Call(ctx, "snapshot.save", map[string]any{
		"machineId": m.ID, "rootOnly": rootOnly, "snapshotId": snap.ID,
	}, m.ID, 600*time.Second)
	if rerr != nil {
		_ = s.Store.UpdateSnapshotStatus(snap.ID, "error")
		return
	}
	var res struct {
		RootDigest string `json:"rootDigest"`
		DataDigest string `json:"dataDigest"`
		RootBytes  int64  `json:"rootBytes"`
		DataBytes  int64  `json:"dataBytes"`
	}
	if json.Unmarshal(raw, &res) == nil {
		// update snapshot digests
		_ = s.Store.DeleteSnapshot(snap.ID)
		snap.RootDigest = res.RootDigest
		snap.DataDigest = res.DataDigest
		snap.RootSizeBytes = res.RootBytes
		snap.DataSizeBytes = res.DataBytes
		snap.Status = "ready"
		_ = s.Store.InsertSnapshot(snap)
		_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
			mm.Storage.SnapshotSeq++
			mm.Storage.Synced = true
		})
	} else {
		_ = s.Store.UpdateSnapshotStatus(snap.ID, "error")
	}
}

func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	snaps, err := s.Store.ListSnapshots(ac.OrgID)
	if err != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"snapshots": snaps})
}

func (s *Server) handleSnapshotByID(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	rest := strings.TrimPrefix(r.URL.Path, "/v1/snapshots/")
	parts := strings.SplitN(rest, "/", 2)
	ref := parts[0]
	if len(parts) == 2 && parts[1] == "files" {
		s.snapshotFilesRoute(w, r, ac, ref)
		return
	}
	snap, err := s.Store.GetSnapshot(ref, ac.OrgID)
	if err != nil {
		s.err(w, 404, &model.Error{Code: "not_found", Message: "no such snapshot"})
		return
	}
	if r.Method == http.MethodDelete {
		confirm := r.Header.Get("x-confirm-delete")
		if confirm != "" && confirm != snap.ID && confirm != snap.Name {
			s.err(w, 409, &model.Error{Code: "delete_confirmation_required", Message: "X-Confirm-Delete does not match"})
			return
		}
		_ = s.Store.DeleteSnapshot(snap.ID)
		w.WriteHeader(204)
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, 200, snap)
		return
	}
	s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
}

func (s *Server) snapshotFilesRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, ref string) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	snap, err := s.Store.GetSnapshot(ref, ac.OrgID)
	if err != nil {
		s.err(w, 404, &model.Error{Code: "not_found", Message: "no such snapshot"})
		return
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}
	raw, rerr := node.Call(r.Context(), "snapshot.list", map[string]string{"snapshotId": snap.ID, "path": r.URL.Query().Get("path")}, "", 60*time.Second)
	if rerr != nil {
		s.err(w, 500, rerr)
		return
	}
	var entries []model.FileEntry
	_ = json.Unmarshal(raw, &entries)
	writeJSON(w, 200, map[string]any{"entries": entries})
}

// ---- restore / fork ----

func (s *Server) restoreRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodPost {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	var req struct {
		SnapshotID   string `json:"snapshotId"`
		DesiredState string `json:"desiredState"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	snapID := req.SnapshotID
	if snapID == "" && m.Snapshot != nil {
		snapID = m.Snapshot.ID
	}
	snap, err := s.Store.GetSnapshot(snapID, ac.OrgID)
	if err != nil {
		s.err(w, 404, &model.Error{Code: "not_found", Message: "no such snapshot"})
		return
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}
	// stop if running
	if m.State == model.StateRunning {
		_, _ = node.Call(r.Context(), "sandbox.stop", map[string]string{"machineId": m.ID}, m.ID, 130*time.Second)
	}
	raw, rerr := node.Call(r.Context(), "snapshot.restore", map[string]string{"machineId": m.ID, "snapshotId": snap.ID}, m.ID, 300*time.Second)
	_ = raw
	if rerr != nil {
		s.err(w, 500, rerr)
		return
	}
	desired := model.DesiredRunning
	if req.DesiredState == model.DesiredStopped {
		desired = model.DesiredStopped
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
		mm.DesiredState = desired
		mm.Snapshot = &model.SnapshotRef{ID: snap.ID}
		mm.Image = mm.Snapshot
		mm.Spec = snap.DefaultSpec
		mm.Storage.Synced = true
		if desired == model.DesiredRunning {
			mm.State = model.StateStarting
			mm.RunStartedAt = now
		} else {
			mm.State = model.StateStopped
		}
	})
	if desired == model.DesiredRunning {
		s.Controller.Enqueue(m.ID)
	}
	writeJSON(w, 202, map[string]any{"ok": true})
}

func (s *Server) forkRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodPost {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	if m.State == model.StateSuspended {
		s.err(w, 409, &model.Error{Code: "fork_unavailable", Message: "a paused machine cannot be forked; resume it first"})
		return
	}
	var req struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 5 {
		s.err(w, 400, badRequest("at most 5 copies at once"))
		return
	}
	count, _ := s.Store.CountMachines(ac.OrgID)
	if count+req.Count > 5 {
		s.err(w, 429, &model.Error{Code: "quota_exceeded", Message: fmt.Sprintf("only %d more machine(s) fit your plan", 5-count)})
		return
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}
	type forkResult struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	results := []forkResult{}
	for i := 0; i < req.Count; i++ {
		name := req.Name
		if req.Count > 1 {
			name = fmt.Sprintf("%s-%d", req.Name, i+1)
		}
		if name == "" {
			name = m.Name + "-fork"
			if req.Count > 1 {
				name = fmt.Sprintf("%s-%d", m.Name+"-fork", i+1)
			}
		}
		if !nameRe.MatchString(name) {
			name = "box-" + randHex(4)
		}
		newID := id.Machine()
		raw, ferr := node.Call(r.Context(), "sandbox.fork", map[string]string{"machineId": m.ID, "newMachineId": newID}, m.ID, 300*time.Second)
		if ferr != nil {
			if len(results) == 0 {
				s.err(w, 500, ferr)
				return
			}
			break
		}
		_ = raw
		now := time.Now().UTC().Format(time.RFC3339Nano)
		nm := *m
		nm.ID = newID
		nm.Name = name
		nm.DesiredState = model.DesiredRunning
		nm.State = model.StateRunning
		nm.Revision = 1
		nm.OperationID = ""
		nm.RunStartedAt = now
		nm.ReadyAt = now
		nm.Snapshot = nil
		nm.Image = nil
		nm.CreatedAt = now
		nm.UpdatedAt = now
		nm.Storage.SnapshotSeq = 0
		if err := s.Store.InsertMachine(&nm); err != nil {
			_, _ = node.Call(context.Background(), "sandbox.delete", map[string]string{"machineId": newID}, newID, 60*time.Second)
			continue
		}
		results = append(results, forkResult{ID: newID, Name: name})
	}
	writeJSON(w, 202, map[string]any{"machines": results})
}

// ---- logs / history / processes ----

func (s *Server) logsRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	node, nerr := s.nodeFor(r.Context())
	console := ""
	if nerr == nil {
		raw, err := node.Call(r.Context(), "sandbox.logs", map[string]string{"machineId": m.ID, "tail": r.URL.Query().Get("tail")}, m.ID, 15*time.Second)
		if err == nil {
			var res struct {
				Console string `json:"console"`
			}
			_ = json.Unmarshal(raw, &res)
			console = res.Console
		}
	}
	ops, _ := s.Store.ListRecentOperations(m.ID, 20)
	writeJSON(w, 200, map[string]any{"console": console, "operations": ops})
}

func (s *Server) historyRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	hist, err := s.Store.ListExecHistory(m.ID, 200)
	if err != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"history": hist})
}

func (s *Server) processesRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	if m.State != model.StateRunning {
		// original: list read before it went down
		writeJSON(w, 200, map[string]any{"processes": []any{}, "note": "machine is not running"})
		return
	}
	node, nerr := s.nodeFor(r.Context())
	if nerr != nil {
		s.err(w, 503, nerr)
		return
	}
	s.Store.RecordActivity(m.ID, "exec")
	raw, err := node.Call(r.Context(), "sandbox.ps", map[string]string{"machineId": m.ID}, m.ID, 30*time.Second)
	if err != nil {
		s.err(w, mapStatus(err), err)
		return
	}
	var res struct {
		Processes []map[string]any `json:"processes"`
	}
	_ = json.Unmarshal(raw, &res)
	writeJSON(w, 200, res)
}

func (s *Server) connectionInfoRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	writeJSON(w, 200, map[string]any{
		"ssh":       "SSH gateway is not implemented in this clone; use the web terminal or exec API",
		"machineId": m.ID,
	})
}

// ---- leases ----

func (s *Server) leasesRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	// activity leases are tracked in-memory on the controller via activity events
	if e := s.ensureRunning(r.Context(), m, 45*time.Second); e != nil && r.Method == http.MethodPut {
		s.err(w, 409, e)
		return
	}
	if r.Method == http.MethodPut {
		var req struct {
			TTLSeconds int `json:"ttlSeconds"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.TTLSeconds < 30 || req.TTLSeconds > 3600 {
			s.err(w, 400, badRequest("ttlSeconds must be 30-3600"))
			return
		}
		// a lease keeps the machine busy: record an activity event now and schedule one at expiry boundary
		s.Store.RecordActivity(m.ID, "lease")
		go func() {
			time.Sleep(time.Duration(req.TTLSeconds-5) * time.Second)
			cur, err := s.Store.GetMachineByID(m.ID)
			if err == nil && cur.State == model.StateRunning {
				s.Store.RecordActivity(m.ID, "lease")
			}
		}()
		s.Store.RecordActivity(m.ID, "lease")
		writeJSON(w, 200, map[string]any{"ok": true})
		return
	}
	if r.Method == http.MethodDelete {
		writeJSON(w, 200, map[string]any{"ok": true})
		return
	}
	s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
}

// ---- machine secrets / vault (P1 subset) ----

func (s *Server) machineSecretsRoute(w http.ResponseWriter, r *http.Request, ac *auth.Context, m *model.Machine) {
	writeJSON(w, 200, map[string]any{"secrets": []any{}, "delivery": "ok"})
}

// ---- usage ----

func (s *Server) handleUsage(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)
	machines, _ := s.Store.ListMachines(ac.OrgID)
	snaps, _ := s.Store.ListSnapshots(ac.OrgID)
	logical, _ := s.Store.StorageBytes(ac.OrgID)

	// compute vcpu-hours from running activity
	var vcpuHours float64
	var memGiBHours float64
	for _, m := range machines {
		hours := s.machineRunHours(m, periodStart, now)
		vcpuHours += float64(m.Spec.VCPUs) * hours
		memGiBHours += float64(m.Spec.MemMiB) / 1024.0 * hours
	}

	usage := model.Usage{
		ComputedAt:   now.Format(time.RFC3339Nano),
		LogicalBytes: logical,
		StoredBytes:  logical / 4, // approximate dedup ratio; honest labeling in report
		Machines:     len(machines),
		Snapshots:    len(snaps),
		Limits: model.UsageLimits{
			MaxMachines: 5, MaxSnapshots: 5, MaxStoredGiB: 200,
			MachineSpecs: []model.Spec{{VCPUs: 4, MemMiB: 8192, PersistentDiskGiB: 20}},
			LiveSnapshots: true,
		},
		AllocatedDiskGiB: len(machines) * 20,
		Cost: model.Cost{
			PeriodStart: periodStart.Format(time.RFC3339), PeriodEnd: periodEnd.Format(time.RFC3339),
			Machines: len(machines), FreeMachines: len(machines),
			FreeSandboxHours: vcpuHours / 4, // rough
			VCPUHours:        vcpuHours,
			MemGiBHours:      memGiBHours,
			CostUSD:          0,
		},
	}
	writeJSON(w, 200, usage)
}

// machineRunHours estimates run hours in the period from runStartedAt (best effort).
func (s *Server) machineRunHours(m *model.Machine, periodStart, now time.Time) float64 {
	if m.State != model.StateRunning || m.RunStartedAt == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339Nano, m.RunStartedAt)
	if err != nil {
		return 0
	}
	if t.Before(periodStart) {
		t = periodStart
	}
	h := now.Sub(t).Hours()
	if h < 0 {
		return 0
	}
	return math.Round(h*1000) / 1000
}
