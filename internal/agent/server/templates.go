// TemplateBuilder lazily materializes template rootfs on the node.
package agentserver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ibox/ibox/internal/agent/sandbox"
)

type TemplateBuilder struct {
	Mgr *sandbox.Manager
	mu  sync.Mutex
}

func NewTemplateBuilder(mgr *sandbox.Manager) *TemplateBuilder {
	return &TemplateBuilder{Mgr: mgr}
}

// dir returns the CAS-extracted template dir.
func (t *TemplateBuilder) dir(id string) string {
	return filepath.Join(t.Mgr.TemplatesDir(), id)
}

// Ensure returns the template rootfs dir, building it if missing.
func (t *TemplateBuilder) Ensure(ctx context.Context, id string) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	dir := t.dir(id)
	if st, err := os.Stat(filepath.Join(dir, "bin")); err == nil && st.IsDir() {
		return dir, nil
	}
	if st, err := os.Stat(filepath.Join(dir, "sbin")); err == nil && st.IsDir() {
		return dir, nil
	}
	os.RemoveAll(dir)
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return "", err
	}
	switch id {
	case "standard":
		return dir, t.buildDebian(ctx, dir, "standard")
	case "minimal":
		return dir, t.buildDebian(ctx, dir, "minimal")
	case "ci":
		return dir, t.buildDebian(ctx, dir, "ci")
	default:
		return "", fmt.Errorf("template %s not available on this node", id)
	}
}

func (t *TemplateBuilder) buildDebian(ctx context.Context, dir, variant string) error {
	pkgs := map[string]string{
		"minimal": "ca-certificates",
		"standard": "ca-certificates,curl,wget,git,jq,vim-tiny,procps,psmisc,iproute2,dnsutils,netcat-openbsd,sudo,build-essential,python3,python3-pip,nodejs,npm",
		"ci":       "ca-certificates,curl,git,jq,build-essential,python3,python3-pip,nodejs,npm,docker.io",
	}
	includes := pkgs[variant]
	cmd := exec.CommandContext(ctx, "debootstrap", "--variant=minbase", "--components=main,contrib,non-free", "--include="+includes, "bookworm", dir, "http://deb.debian.org/debian")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// retry with default mirror (some environments block http)
		cmd = exec.CommandContext(ctx, "debootstrap", "--variant=minbase", "--include="+includes, "bookworm", dir)
		out, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("debootstrap: %v: %s", err, tail(out))
		}
	}
	// in-chroot finishing: create user, sudo, hostname bits
	chrootSh := `
set -e
useradd -m -s /bin/bash user 2>/dev/null || true
mkdir -p /etc/sudoers.d /data /run/ibox/env
echo 'user ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/user
chmod 0440 /etc/sudoers.d/user
chown user:user /data 2>/dev/null || true
echo 'export PATH=$PATH:/usr/local/bin' >> /etc/profile
echo '127.0.0.1 localhost' > /etc/hosts
`
	if variant != "minimal" {
		chrootSh += `
echo 'nameserver 10.0.2.3' > /etc/resolv.conf 2>/dev/null || true
`
	}
	c := exec.CommandContext(ctx, "chroot", dir, "/bin/sh", "-c", chrootSh)
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("chroot setup: %v: %s", err, tail(out))
	}
	return nil
}

func tail(b []byte) string {
	s := string(b)
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	return strings.Join(lines, "\n")
}
