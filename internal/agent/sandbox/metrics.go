// Metrics sampling from cgroup v2.
package sandbox

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ibox/ibox/internal/model"
)

// Sample collects a metric sample for a sandbox.
func (m *Manager) Sample(sb *Sandbox) *model.MetricSample {
	now := time.Now().UTC()
	smp := &model.MetricSample{At: now, MemTotalBytes: int64(sbMemLimit(sb))}
	cg := m.cgroupPath(sb)
	if cg == "" {
		return smp
	}
	// memory
	if b, err := os.ReadFile(filepath.Join(cg, "memory.current")); err == nil {
		smp.MemUsedBytes, _ = strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	}
	if b, err := os.ReadFile(filepath.Join(cg, "memory.max")); err == nil {
		if v, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil && v > 0 {
			smp.MemTotalBytes = v
		}
	}
	// cpu usage (nanoseconds)
	if b, err := os.ReadFile(filepath.Join(cg, "cpu.stat")); err == nil {
		usage := uint64(0)
		for _, line := range strings.Split(string(b), "\n") {
			f := strings.Fields(line)
			if len(f) == 2 && f[0] == "usage_usec" {
				v, _ := strconv.ParseUint(f[1], 10, 64)
				usage = v * 1000
			}
		}
		sb.mu.Lock()
		prev := sb.lastCPUTime
		sb.lastCPUTime = usage
		sb.mu.Unlock()
		dt := uint64(now.Sub(sb.lastSample).Nanoseconds())
		if dt > 0 && prev > 0 && usage >= prev {
			// percent of 1 cpu
			smp.CPUPercent = float64(usage-prev) * 100 / float64(dt)
		}
	}
	// io
	if b, err := os.ReadFile(filepath.Join(cg, "io.stat")); err == nil {
		var r, w uint64
		for _, line := range strings.Split(string(b), "\n") {
			if !strings.HasPrefix(line, "8:") && !strings.HasPrefix(line, "259:") {
				continue
			}
			for _, f := range strings.Fields(line) {
				if strings.HasPrefix(f, "rbytes=") {
					v, _ := strconv.ParseUint(strings.TrimPrefix(f, "rbytes="), 10, 64)
					r += v
				}
				if strings.HasPrefix(f, "wbytes=") {
					v, _ := strconv.ParseUint(strings.TrimPrefix(f, "wbytes="), 10, 64)
					w += v
				}
			}
		}
		sb.mu.Lock()
		pr, pw := sb.lastIO.rbytes, sb.lastIO.wbytes
		sb.lastIO = ioCounters{rbytes: r, wbytes: w}
		sb.lastSample = now
		sb.mu.Unlock()
		if r >= pr {
			smp.DiskRBytes = int64(r - pr)
		}
		if w >= pw {
			smp.DiskWBytes = int64(w - pw)
		}
	}
	// network: not tracked per-cgroup without nft hooks; reported as 0 (excludes SSH/ports, like original)
	return smp
}

// sbMemLimit is a placeholder when memory.max is missing.
func sbMemLimit(sb *Sandbox) int { return 0 }

// cgroupPath finds the sandbox's cgroup v2 directory.
func (m *Manager) cgroupPath(sb *Sandbox) string {
	if sb.cgroupDir != "" {
		if _, err := os.Stat(sb.cgroupDir); err == nil {
			return sb.cgroupDir
		}
	}
	// search common runsc/runc cgroup layouts
	roots := []string{
		"/sys/fs/cgroup",
	}
	name := sb.MachineID
	for _, root := range roots {
		// containerd/runc style: /sys/fs/cgroup/<subsystem?>... cgroup v2 unified: find by walking
		hits := []string{}
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir
			}
			if !d.IsDir() {
				return nil
			}
			if strings.HasSuffix(p, name) {
				hits = append(hits, p)
				return filepath.SkipAll
			}
			// prune: skip too deep early-exit
			if rel, _ := filepath.Rel(root, p); rel != "." && strings.Count(rel, string(filepath.Separator)) > 8 {
				return filepath.SkipDir
			}
			return nil
		})
		if len(hits) > 0 {
			sb.cgroupDir = hits[0]
			return hits[0]
		}
	}
	return ""
}

// AllSamples collects samples for all running sandboxes.
func (m *Manager) AllSamples() map[string]*model.MetricSample {
	out := map[string]*model.MetricSample{}
	for _, sb := range m.List() {
		if sb.State != StateRunning {
			continue
		}
		out[sb.MachineID] = m.Sample(sb)
	}
	return out
}
