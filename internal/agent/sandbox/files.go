// Host-side file access for sandboxes: the rootfs and data dirs are real
// directories on the node, so listing/reading/writing is direct I/O.
package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ibox/ibox/internal/model"
)

// ErrMissingParent mirrors the original product's 404 missing_parent.
var ErrMissingParent = errors.New("missing_parent")
var ErrNotFound = errors.New("not_found")

// sandboxPath maps an in-guest absolute path to the host path.
// /data maps to the data volume; everything else to the rootfs.
func (sb *Sandbox) hostPath(guestPath string) (string, error) {
	if guestPath == "" {
		guestPath = "/"
	}
	if !strings.HasPrefix(guestPath, "/") {
		return "", errors.New("path must be absolute")
	}
	clean := filepath.Clean(guestPath)
	if clean == "/data" || strings.HasPrefix(clean, "/data/") {
		rel := strings.TrimPrefix(clean, "/data")
		return filepath.Join(sb.DataDir, rel), nil
	}
	return filepath.Join(sb.RootDir, clean), nil
}

// List lists a directory of a sandbox.
func (sb *Sandbox) List(guestPath string) ([]model.FileEntry, error) {
	host, err := sb.hostPath(guestPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(host)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !info.IsDir() {
		// reading a file via list returns single entry
		return []model.FileEntry{fileEntry(host, guestPath, info)}, nil
	}
	entries, err := os.ReadDir(host)
	if err != nil {
		return nil, err
	}
	out := make([]model.FileEntry, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			// dangling symlink etc.
			fi, _ = os.Lstat(filepath.Join(host, e.Name()))
			if fi == nil {
				continue
			}
		}
		p := guestPath
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		out = append(out, fileEntry(filepath.Join(host, e.Name()), p+e.Name(), fi))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func fileEntry(hostPath, guestPath string, info os.FileInfo) model.FileEntry {
	e := model.FileEntry{
		Name:    info.Name(),
		Path:    guestPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime().UTC().Format(time.RFC3339),
	}
	if info.Mode()&os.ModeSymlink != 0 {
		e.IsLink = true
		if target, err := os.Readlink(hostPath); err == nil {
			e.LinkTo = target
		}
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		_ = st
	}
	return e
}

// ReadFile reads a file's bytes.
func (sb *Sandbox) ReadFile(guestPath string) ([]byte, error) {
	host, err := sb.hostPath(guestPath)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(host)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

// WriteFile writes bytes, optionally creating parents.
func (sb *Sandbox) WriteFile(guestPath string, content []byte, parents bool) error {
	host, err := sb.hostPath(guestPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(host)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if !parents {
			return ErrMissingParent
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := host + ".ibox-tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	// preserve mode if file exists
	if info, err := os.Stat(host); err == nil {
		os.Chmod(tmp, info.Mode())
	}
	return os.Rename(tmp, host)
}

// DeletePath removes a file or empty directory.
func (sb *Sandbox) DeletePath(guestPath string) error {
	host, err := sb.hostPath(guestPath)
	if err != nil {
		return err
	}
	if err := os.Remove(host); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Mkdir creates a directory with parents.
func (sb *Sandbox) Mkdir(guestPath string) error {
	host, err := sb.hostPath(guestPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(host, 0o755)
}

// Stat returns file info string for logs.
func (sb *Sandbox) String() string {
	return fmt.Sprintf("sandbox %s (%s)", sb.MachineID, sb.State)
}
