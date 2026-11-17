// Exec runs commands inside sandboxes via runsc/runc exec.
package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// ExecResult is the outcome of a foreground exec.
type ExecResult struct {
	ExitCode  int
	Stdout    []byte
	Stderr    []byte
	Truncated bool
}

// Exec runs argv inside the sandbox with optional timeout.
// Shell=true runs argv[0] through /bin/sh -c.
func (m *Manager) Exec(ctx context.Context, sb *Sandbox, argv []string, cwd string, env map[string]string, stdin []byte, timeoutS int, shell bool) (*ExecResult, error) {
	if sb.State != StateRunning {
		return nil, errors.New("machine_not_running")
	}
	if len(argv) == 0 {
		return nil, errors.New("empty command")
	}
	if cwd == "" {
		cwd = "/data/home/user"
	}

	var args []string
	if sb.Runtime == RuntimeRunsc && sb.Network == "host" {
		args = append(args, "--network=host")
	}
	args = append(args, "--root", m.runscRoot(sb.Runtime), "exec")
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	for k, v := range env {
		args = append(args, "--env", k+"="+v)
	}
	cmdStr := argv[0]
	if shell {
		args = append(args, "--", sb.MachineID, "/bin/sh", "-c", cmdStr)
	} else {
		args = append(args, "--", sb.MachineID)
		args = append(args, argv...)
	}

	cmd := exec.CommandContext(ctx, sb.Runtime, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	// cap memory of output buffers
	stdout.Grow(1 << 20)
	stderr.Grow(1 << 20)
	cmd.Stdout = &capWriter{b: &stdout, cap: 8 << 20}
	cmd.Stderr = &capWriter{b: &stderr, cap: 8 << 20}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	start := time.Now()
	if timeoutS > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutS)*time.Second)
		defer cancel()
		cmd = exec.CommandContext(ctx, sb.Runtime, args...)
		if len(stdin) > 0 {
			cmd.Stdin = bytes.NewReader(stdin)
		}
		stdout.Reset()
		stderr.Reset()
		cmd.Stdout = &capWriter{b: &stdout, cap: 8 << 20}
		cmd.Stderr = &capWriter{b: &stderr, cap: 8 << 20}
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	err := cmd.Run()
	_ = start
	res := &ExecResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	truncated := false
	if len(res.Stdout) > 1<<20 {
		res.Stdout = res.Stdout[:1<<20]
		truncated = true
	}
	if len(res.Stderr) > 1<<20 {
		res.Stderr = res.Stderr[:1<<20]
		truncated = true
	}
	res.Truncated = truncated

	if err == nil {
		res.ExitCode = 0
		return res, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		res.ExitCode = ee.ExitCode()
		if res.ExitCode == -1 && ctx.Err() == context.DeadlineExceeded {
			res.ExitCode = 124
		}
		return res, nil
	}
	// context cancellation → treat as timeout kill (124)
	if ctx != nil && ctx.Err() != nil {
		res.ExitCode = 124
		return res, nil
	}
	return res, fmt.Errorf("exec: %w", err)
}

type capWriter struct {
	b   *bytes.Buffer
	cap int
}

func (w *capWriter) Write(p []byte) (int, error) {
	if w.b.Len() >= w.cap {
		return len(p), nil
	}
	remaining := w.cap - w.b.Len()
	if len(p) > remaining {
		p = p[:remaining]
	}
	return w.b.Write(p)
}

// ExecPTY opens an interactive PTY exec. Returns the started command; caller
// pipes stdin/stdout through the returned file (pty master).
func (m *Manager) ExecPTY(ctx context.Context, sb *Sandbox, shell string) (*os.File, error) {
	if sb.State != StateRunning {
		return nil, errors.New("machine_not_running")
	}
	if shell == "" {
		shell = "/bin/bash"
	}
	ptmx, pts, err := openPTY()
	if err != nil {
		return nil, err
	}

	var args []string
	if sb.Runtime == RuntimeRunsc && sb.Network == "host" {
		args = append(args, "--network=host")
	}
	// NOTE: runsc exec has no -t flag; passing the PTY slave as stdio gives
	// the executed process a real terminal (isatty(0) == true).
	args = append(args, "--root", m.runscRoot(sb.Runtime), "exec", "--", sb.MachineID, shell)
	cmd := exec.CommandContext(ctx, sb.Runtime, args...)
	cmd.Stdin = pts
	cmd.Stdout = pts
	cmd.Stderr = pts
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		ptmx.Close()
		pts.Close()
		return nil, err
	}
	pts.Close() // child holds its own copy
	go func() {
		cmd.Wait()
		ptmx.Close()
	}()
	return ptmx, nil
}

// openPTY creates a pty master/slave pair.
func openPTY() (master *os.File, slave *os.File, err error) {
	return openPTYImpl()
}

// Process is one line of ps output.
type Process struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	User    string `json:"user"`
	Command string `json:"command"`
}

// PS lists processes inside the sandbox.
func (m *Manager) PS(sb *Sandbox) ([]Process, error) {
	if sb.State != StateRunning {
		return nil, errors.New("machine_not_running")
	}
	res, err := m.Exec(context.Background(), sb, []string{"ps", "-eo", "pid,ppid,user,args", "--no-headers"}, "/", nil, nil, 30, false)
	if err != nil {
		return nil, err
	}
	var out []Process
	for _, line := range strings.Split(string(res.Stdout), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		var p Process
		fmt.Sscanf(fields[0], "%d", &p.PID)
		fmt.Sscanf(fields[1], "%d", &p.PPID)
		p.User = fields[2]
		p.Command = strings.Join(fields[3:], " ")
		out = append(out, p)
	}
	return out, nil
}

var _ = json.Marshal
var _ = filepath.Join
