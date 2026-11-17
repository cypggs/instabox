// Package controller executes machine operations against node agents and
// drives auto-suspend, max-run cooldown, and deadlines.
package controller

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/ibox/ibox/internal/id"
	"github.com/ibox/ibox/internal/model"
	"github.com/ibox/ibox/internal/nodes"
	"github.com/ibox/ibox/internal/store"
)

type Controller struct {
	Store    *store.Store
	Registry *nodes.Registry

	wake chan string // machine IDs needing operation dispatch
	mu   sync.Mutex
	busy map[string]bool

	// config
	IdleDefaultSeconds  int
	MaxRunStartMinutes  int
	MaxRunCapMinutes    int
	FreeTier            bool
}

func New(st *store.Store, reg *nodes.Registry) *Controller {
	return &Controller{
		Store: st, Registry: reg,
		wake: make(chan string, 256),
		busy: map[string]bool{},
		IdleDefaultSeconds: 1800,
		MaxRunStartMinutes: 120,
		MaxRunCapMinutes:   480,
		FreeTier:           true,
	}
}

// Enqueue marks a machine for operation dispatch.
func (c *Controller) Enqueue(machineID string) {
	select {
	case c.wake <- machineID:
	default:
	}
}

// Run starts background loops. Blocks until ctx is cancelled.
func (c *Controller) Run(ctx context.Context) {
	go c.dispatchLoop(ctx)
	go c.reaperLoop(ctx)
	go c.metricLoop(ctx)
	<-ctx.Done()
}

// dispatchLoop picks up machines whose desired != actual state.
func (c *Controller) dispatchLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case mid := <-c.wake:
			c.reconcileMachine(ctx, mid)
		case <-ticker.C:
			c.reconcileAll(ctx)
		}
	}
}

func (c *Controller) reconcileAll(ctx context.Context) {
	// scan all machines; we need org list — use a special org sweep
	orgs, err := c.listOrgIDs()
	if err != nil {
		return
	}
	for _, org := range orgs {
		ms, err := c.Store.ListMachines(org)
		if err != nil {
			continue
		}
		for _, m := range ms {
			if c.needsReconcile(m) {
				c.reconcileMachine(ctx, m.ID)
			}
		}
	}
}

func (c *Controller) listOrgIDs() ([]string, error) {
	return c.Store.ListOrgIDs()
}

func (c *Controller) needsReconcile(m *model.Machine) bool {
	// a recorded operation may be stale (crashed owner); let reconcileMachine
	// verify and clean it up
	if m.OperationID != "" {
		return true
	}
	switch m.DesiredState {
	case model.DesiredRunning:
		return m.State != model.StateRunning && m.State != model.StateStarting && m.State != model.StateResuming
	case model.DesiredStopped:
		return m.State != model.StateStopped && m.State != model.StateStopping
	case model.DesiredSuspended:
		return m.State != model.StateSuspended && m.State != model.StateSuspending
	case model.DesiredDeleted:
		// re-enter a stuck deleting state (previous delete op crashed)
		return m.State != model.StateDeleting || m.OperationID == ""
	}
	return false
}

// reconcileMachine drives one machine toward desired state via operations.
// staleOperation reports whether a running operation has had no heartbeat
// long enough that its owner must have crashed (control plane restart, node
// disconnect). StartedAt is RFC3339.
func staleOperation(op *model.Operation) bool {
	if op.StartedAt == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339Nano, op.StartedAt)
	if err != nil {
		return false
	}
	return time.Since(t) > 5*time.Minute
}

func (c *Controller) reconcileMachine(ctx context.Context, machineID string) {
	c.mu.Lock()
	if c.busy[machineID] {
		c.mu.Unlock()
		return
	}
	c.busy[machineID] = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.busy, machineID)
		c.mu.Unlock()
	}()

	m, err := c.Store.GetMachineByID(machineID)
	if err != nil {
		return
	}
	if m.OperationID != "" {
		// an operation is recorded; verify it is still being processed
		op, err := c.Store.GetOperation(m.OperationID)
		if err == nil && (op.Status == "pending" || op.Status == "running") {
			if op.Status == "running" && staleOperation(op) {
				// the control plane crashed or the node vanished mid-operation;
				// fail it and let reconciliation retry the transition
				_ = c.Store.FinishOperation(op.ID, "failed", &model.Error{Code: "interrupted", Message: "operation interrupted by control-plane restart"}, nil)
				_ = c.Store.ReleaseMachineLock(machineID)
				_ = c.Store.UpdateMachine(machineID, func(mm *model.Machine) { mm.OperationID = "" })
				m.OperationID = ""
			} else {
				return // in flight
			}
		} else {
			// stale lock; release
			_ = c.Store.ReleaseMachineLock(machineID)
			_ = c.Store.UpdateMachine(machineID, func(mm *model.Machine) { mm.OperationID = "" })
			m.OperationID = ""
		}
	}

	// deleted?
	if m.DesiredState == model.DesiredDeleted && (m.State != model.StateDeleting || m.OperationID == "") {
		c.runOperation(ctx, m, "delete", nil, c.opDelete)
		return
	}

	// cooldown / not-running handling for transitions that need a node
	node, err := c.Registry.AnyNode()
	if err != nil {
		return // no node yet; retry next tick
	}

	transition := ""
	switch {
	case m.DesiredState == model.DesiredRunning && m.State == model.StateStopped:
		transition = "start"
	case m.DesiredState == model.DesiredRunning && m.State == model.StateSuspended:
		transition = "resume"
	case m.DesiredState == model.DesiredStopped && m.State == model.StateRunning:
		transition = "stop"
	case m.DesiredState == model.DesiredStopped && m.State == model.StateSuspended:
		// cannot stop a suspended machine; resume first (matches original product)
		transition = "resume"
	case m.DesiredState == model.DesiredSuspended && m.State == model.StateRunning:
		transition = "suspend"
	case m.DesiredState == model.DesiredRunning && m.State == model.StateStarting:
		// previous start was interrupted (crash / RPC timeout) — retry
		transition = "start"
	case m.DesiredState == model.DesiredRunning && m.State == model.StateError:
		transition = "start"
	}
	if transition == "" {
		return
	}

	if c.inCooldown(m) {
		return
	}

	switch transition {
	case "start":
		c.runOperation(ctx, m, "start", nil, c.opStart(node))
	case "resume":
		c.runOperation(ctx, m, "resume", nil, c.opResume(node))
	case "stop":
		c.runOperation(ctx, m, "stop", nil, c.opStop(node))
	case "suspend":
		c.runOperation(ctx, m, "suspend", nil, c.opSuspend(node))
	}
}

func (c *Controller) inCooldown(m *model.Machine) bool {
	if m.CooldownUntil == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339Nano, m.CooldownUntil)
	return err == nil && time.Now().UTC().Before(t)
}

type opFunc func(ctx context.Context, m *model.Machine) *model.Error

// runOperation wraps an operation with locking and status updates.
func (c *Controller) runOperation(ctx context.Context, m *model.Machine, kind string, req any, fn opFunc) {
	opID := id.Operation()
	op := &model.Operation{ID: opID, MachineID: m.ID, Kind: kind, Status: "pending", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if req != nil {
		b, _ := json.Marshal(req)
		_ = b
	}
	if err := c.Store.InsertOperation(op); err != nil {
		log.Printf("insert operation: %v", err)
		return
	}
	ok, err := c.Store.AcquireMachineLock(m.ID, opID)
	if err != nil || !ok {
		return
	}
	_ = c.Store.SetOperationRunning(opID)
	_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.OperationID = opID })

	// state transition marker
	markState := ""
	switch kind {
	case "start":
		markState = model.StateStarting
	case "resume":
		markState = model.StateResuming
	case "stop":
		markState = model.StateStopping
	case "suspend":
		markState = model.StateSuspending
	case "delete":
		markState = model.StateDeleting
	}
	if markState != "" {
		_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.State = markState })
	}

	opErr := fn(ctx, m)
	if opErr == nil {
		_ = c.Store.FinishOperation(opID, "succeeded", nil, nil)
	} else {
		log.Printf("operation %s %s failed: %s: %s", kind, m.ID, opErr.Code, opErr.Message)
		_ = c.Store.FinishOperation(opID, "failed", opErr, nil)
		_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
			if kind != "delete" {
				mm.State = model.StateError
			}
		})
	}
	_ = c.Store.ReleaseMachineLock(m.ID)
	_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.OperationID = "" })
}

// ---- operations ----

func (c *Controller) opStart(node *nodes.Node) opFunc {
	return func(ctx context.Context, m *model.Machine) *model.Error {
		_, opErr := node.Call(ctx, "sandbox.start", nodes.SandboxSpec{
			MachineID: m.ID, Name: m.Name,
			VCPUs: m.Spec.VCPUs, MemMiB: m.Spec.MemMiB,
			RootDiskGiB: m.Spec.RootDiskGiB, DiskGiB: m.Spec.PersistentDiskGiB,
			GuestUser: m.GuestUser,
		}, m.ID, 120*time.Second)
		if opErr != nil && opErr.Code == "not_found" {
			// the machine was never provisioned on the node (e.g. the create
			// operation timed out or the node was rebuilt) — re-create it
			createParams := map[string]any{
				"spec": nodes.SandboxSpec{
					MachineID: m.ID, Name: m.Name,
					VCPUs: m.Spec.VCPUs, MemMiB: m.Spec.MemMiB,
					RootDiskGiB: m.Spec.RootDiskGiB, DiskGiB: m.Spec.PersistentDiskGiB,
					GuestUser: m.GuestUser,
				},
			}
			if m.Template != nil {
				createParams["templateId"] = m.Template.ID
			}
			_, opErr = node.Call(ctx, "sandbox.create", createParams, m.ID, 20*time.Minute)
			if opErr != nil {
				return opErr
			}
			_, opErr = node.Call(ctx, "sandbox.start", nodes.SandboxSpec{
				MachineID: m.ID, Name: m.Name,
				VCPUs: m.Spec.VCPUs, MemMiB: m.Spec.MemMiB,
				RootDiskGiB: m.Spec.RootDiskGiB, DiskGiB: m.Spec.PersistentDiskGiB,
				GuestUser: m.GuestUser,
			}, m.ID, 120*time.Second)
		}
		if opErr != nil {
			return opErr
		}
		_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
			mm.State = model.StateRunning
			mm.NodeID = node.ID
			mm.RunStartedAt = time.Now().UTC().Format(time.RFC3339Nano)
			if mm.ReadyAt == "" {
				mm.ReadyAt = time.Now().UTC().Format(time.RFC3339Nano)
			}
			mm.AutoSuspendedAt = ""
		})
		c.activity(m.ID, "start")
		return nil
	}
}

func (c *Controller) opResume(node *nodes.Node) opFunc {
	return func(ctx context.Context, m *model.Machine) *model.Error {
		res, opErr := node.Call(ctx, "sandbox.resume", map[string]string{"machineId": m.ID}, m.ID, 180*time.Second)
		if opErr != nil {
			// resume failed → fresh start with files (product semantics)
			log.Printf("resume %s failed (%s); starting fresh", m.ID, opErr.Code)
			_, opErr2 := node.Call(ctx, "sandbox.start", nodes.SandboxSpec{
				MachineID: m.ID, Name: m.Name,
				VCPUs: m.Spec.VCPUs, MemMiB: m.Spec.MemMiB,
				RootDiskGiB: m.Spec.RootDiskGiB, DiskGiB: m.Spec.PersistentDiskGiB,
				GuestUser: m.GuestUser,
			}, m.ID, 120*time.Second)
			if opErr2 != nil {
				return opErr2
			}
		} else {
			var info struct {
				Restored bool `json:"restored"`
			}
			_ = json.Unmarshal(res, &info)
			_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
				if !info.Restored {
					mm.Runtime.ProcessesPreserved = false
				}
			})
		}
		_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
			mm.State = model.StateRunning
			mm.NodeID = node.ID
			mm.RunStartedAt = time.Now().UTC().Format(time.RFC3339Nano)
			mm.ReadyAt = time.Now().UTC().Format(time.RFC3339Nano)
			mm.AutoSuspendedAt = ""
		})
		c.activity(m.ID, "resume")
		return nil
	}
}

func (c *Controller) opStop(node *nodes.Node) opFunc {
	return func(ctx context.Context, m *model.Machine) *model.Error {
		_, opErr := node.Call(ctx, "sandbox.stop", map[string]string{"machineId": m.ID}, m.ID, 130*time.Second)
		if opErr != nil && opErr.Code != "machine_not_running" {
			return opErr
		}
		_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
			mm.State = model.StateStopped
			mm.RunStartedAt = ""
			mm.ReadyAt = ""
		})
		return nil
	}
}

func (c *Controller) opSuspend(node *nodes.Node) opFunc {
	return func(ctx context.Context, m *model.Machine) *model.Error {
		_, opErr := node.Call(ctx, "sandbox.suspend", map[string]string{"machineId": m.ID}, m.ID, 180*time.Second)
		if opErr != nil {
			return opErr
		}
		_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
			mm.State = model.StateSuspended
			mm.Storage.Synced = true
			mm.RunStartedAt = ""
		})
		return nil
	}
}

func (c *Controller) opDelete(ctx context.Context, m *model.Machine) *model.Error {
	if node, err := c.Registry.AnyNode(); err == nil {
		_, _ = node.Call(ctx, "sandbox.delete", map[string]string{"machineId": m.ID}, m.ID, 120*time.Second)
	}
	if err := c.Store.DeleteMachine(m.ID); err != nil {
		log.Printf("delete %s: store removal failed: %v (will retry)", m.ID, err)
		return &model.Error{Code: "internal", Message: err.Error(), Retryable: true}
	}
	return nil
}

// ---- background loops ----

// reaperLoop: auto-suspend idle machines, enforce max-run + cooldown, deadlines.
func (c *Controller) reaperLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkMachines(ctx)
		}
	}
}

func (c *Controller) checkMachines(ctx context.Context) {
	orgs, err := c.listOrgIDs()
	if err != nil {
		return
	}
	for _, org := range orgs {
		ms, err := c.Store.ListMachines(org)
		if err != nil {
			continue
		}
		for _, m := range ms {
			c.checkDeadlines(m)
			if m.State != model.StateRunning {
				continue
			}
			now := time.Now().UTC()
			// max-run enforcement (free plan)
			if c.FreeTier && m.MaxRunMinutes > 0 && m.RunStartedAt != "" {
				if rs, err := time.Parse(time.RFC3339Nano, m.RunStartedAt); err == nil {
					runEnd := rs.Add(time.Duration(m.MaxRunMinutes) * time.Minute)
					if now.After(runEnd) {
						c.maxRunPause(m)
						continue
					}
					// extension: with 5 minutes left, decide based on recent usage
					if now.After(runEnd.Add(-5 * time.Minute)) && m.MaxRunMinutes < c.MaxRunCapMinutes {
						used := c.minutesUsed(m.ID, 30*time.Minute)
						grow := 0
						if used >= 5 {
							grow = 120
						} else if used >= 1 {
							grow = 30
						}
						if grow > 0 {
							newMax := m.MaxRunMinutes + grow
							if newMax > c.MaxRunCapMinutes {
								newMax = c.MaxRunCapMinutes
							}
							_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.MaxRunMinutes = newMax })
						}
					}
				}
			}
			// auto-suspend on idle
			if m.AutoSuspendAfterSeconds > 0 {
				last, err := c.Store.LatestActivity(m.ID)
				if err != nil || last.IsZero() {
					// no activity recorded since start; use runStartedAt
					if rs, err2 := time.Parse(time.RFC3339Nano, m.RunStartedAt); err2 == nil {
						last = rs
					}
				}
				if now.Sub(last) > time.Duration(m.AutoSuspendAfterSeconds)*time.Second {
					log.Printf("auto-suspending idle machine %s", m.Name)
					_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
						mm.DesiredState = model.DesiredSuspended
					})
					_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
						mm.AutoSuspendedAt = now.Format(time.RFC3339Nano)
					})
					c.Enqueue(m.ID)
				}
			}
		}
	}
}

func (c *Controller) maxRunPause(m *model.Machine) {
	cooldownBase := 5 * time.Minute
	// count consecutive max-run pauses via metadata
	streak := 0
	if m.Metadata != nil {
		if s, ok := m.Metadata["cooldownStreak"]; ok {
			if n, err := strconv.Atoi(s); err == nil {
				streak = n
			}
		}
	}
	if streak == 1 {
		cooldownBase = 60 * time.Minute
	} else if streak >= 2 {
		cooldownBase = 120 * time.Minute
	}
	cooldown := cooldownBase + time.Duration(rand.Intn(15))*time.Minute
	until := time.Now().UTC().Add(cooldown).Format(time.RFC3339Nano)
	_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
		mm.DesiredState = model.DesiredSuspended
		mm.CooldownUntil = until
		mm.MaxRunMinutes = c.MaxRunCapMinutes // show cap when paused, like the original
		mm.AutoSuspendedAt = time.Now().UTC().Format(time.RFC3339Nano)
		mm.Metadata["cooldownStreak"] = itoa(streak + 1)
	})
	c.Enqueue(m.ID)
}

func fmtStr(s string, out *int) (int, error) {
	n, err := strconv.Atoi(s)
	*out = n
	return n, err
}

func itoa(n int) string { return strconv.Itoa(n) }

func (c *Controller) minutesUsed(machineID string, window time.Duration) int {
	n, err := c.Store.CountActivitySince(machineID, time.Now().UTC().Add(-window), "cpu")
	if err != nil {
		return 0
	}
	// each cpu event represents a 30s sample
	return n / 2
}

func (c *Controller) checkDeadlines(m *model.Machine) {
	now := time.Now().UTC()
	if m.DeleteAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, m.DeleteAt); err == nil && now.After(t) {
			_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.DesiredState = model.DesiredDeleted })
			c.Enqueue(m.ID)
			return
		}
	}
	if m.ArchiveAt != "" && m.State == model.StateRunning {
		if t, err := time.Parse(time.RFC3339Nano, m.ArchiveAt); err == nil && now.After(t) {
			_ = c.Store.UpdateMachine(m.ID, func(mm *model.Machine) {
				mm.DesiredState = model.DesiredStopped
				mm.ArchiveAt = ""
			})
			c.Enqueue(m.ID)
		}
	}
}

// metricLoop pulls metrics from nodes every 30s.
func (c *Controller) metricLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			node, err := c.Registry.AnyNode()
			if err != nil {
				continue
			}
			res, opErr := node.Call(ctx, "sandbox.metrics", nil, "", 30*time.Second)
			if opErr != nil {
				continue
			}
			var samples map[string]*model.MetricSample // machineId -> sample
			if json.Unmarshal(res, &samples) != nil {
				continue
			}
			for mid, smp := range samples {
				if smp == nil {
					continue
				}
				_ = c.Store.InsertMetric(mid, smp)
				// cpu activity: >=25% of one vCPU counts as usage
				if smp.CPUPercent >= 25.0 {
					c.activity(mid, "cpu")
				}
			}
		}
	}
}

func (c *Controller) activity(machineID, kind string) {
	_ = c.Store.RecordActivity(machineID, kind)
}

// HandleEvent processes async agent events.
func (c *Controller) HandleEvent(ev *nodes.Event) {
	if ev.Event == "machine.state" && ev.Machine != "" && ev.State != "" {
		_ = c.Store.UpdateMachine(ev.Machine, func(m *model.Machine) {
			// only adopt states that make sense from agent
			switch ev.State {
			case model.StateRunning, model.StateStopped, model.StateSuspended:
				m.State = ev.State
			}
		})
	}
}
