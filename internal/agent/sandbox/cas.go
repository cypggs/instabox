// CAS (content-addressed storage) for template rootfs and snapshots.
package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ibox/ibox/internal/model"
)

// CASDir is the content-addressed store root.
func (m *Manager) CASDir() string {
	return filepath.Join(m.BaseDir, "cas")
}

func casPath(root, digest string) string {
	d := strings.TrimPrefix(digest, "sha256:")
	return filepath.Join(root, d[:2], d[2:4], d)
}

// SaveSnapshot archives a sandbox's rootfs (+ data unless rootOnly) into CAS.
func (m *Manager) SaveSnapshot(sb *Sandbox, rootOnly bool) (rootDigest, dataDigest string, rootBytes, dataBytes int64, err error) {
	if sb.State == StateSuspended || sb.State == StatePaused {
		return "", "", 0, 0, fmt.Errorf("machine is paused")
	}
	casRoot := m.CASDir()
	os.MkdirAll(casRoot, 0o755)

	rootDigest, rootBytes, err = archiveDir(casRoot, sb.RootDir)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("archive root: %w", err)
	}
	if !rootOnly {
		dataDigest, dataBytes, err = archiveDir(casRoot, sb.DataDir)
		if err != nil {
			return "", "", 0, 0, fmt.Errorf("archive data: %w", err)
		}
	}
	return rootDigest, dataDigest, rootBytes, dataBytes, nil
}

// archiveDir tars+zstd-compresses dir into CAS, returning sha256 digest and size.
func archiveDir(casRoot, dir string) (string, int64, error) {
	tmp, err := os.CreateTemp("", "ibox-cas-*.tar.zst")
	if err != nil {
		return "", 0, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	// tar -C dir -cf - . | zstd -q -T0 -3 > tmp
	// .gvisor.* files are the sandbox's overlay filestore metadata — archiving
	// them poisons the snapshot: a restored rootfs makes runsc refuse to start
	// ("mount source already has a filestore file").
	sh := exec.Command("sh", "-c", fmt.Sprintf("tar -C %s --exclude='.gvisor.*' --exclude='*/.gvisor.*' -cf - . | zstd -q -T0 -3 > %s", shellQuote(dir), shellQuote(tmpName)))
	if out, err := sh.CombinedOutput(); err != nil {
		tmp.Close()
		return "", 0, fmt.Errorf("archive: %v: %s", err, out)
	}
	tmp.Close()

	sumB, err := exec.Command("sha256sum", tmpName).Output()
	if err != nil {
		return "", 0, err
	}
	sum := strings.Fields(string(sumB))[0]
	dst := casPath(casRoot, sum)
	os.MkdirAll(filepath.Dir(dst), 0o755)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		if err := os.Rename(tmpName, dst); err != nil {
			return "", 0, err
		}
	}
	info, err := os.Stat(dst)
	if err != nil {
		return "", 0, err
	}
	return "sha256:" + sum, info.Size(), nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ExtractSnapshot unpacks a snapshot's CAS archives to a directory.
func (m *Manager) ExtractSnapshot(rootDigest, dataDigest, dst string) error {
	casRoot := m.CASDir()
	for _, digest := range []string{rootDigest, dataDigest} {
		if digest == "" {
			continue
		}
		src := casPath(casRoot, digest)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("snapshot content not found on this node: %s", digest)
		}
	}
	// root into dst/root, data into dst/data
	if rootDigest != "" {
		if err := extractArchive(casRoot, rootDigest, filepath.Join(dst, "root-extract")); err != nil {
			return err
		}
	}
	if dataDigest != "" {
		if err := extractArchive(casRoot, dataDigest, filepath.Join(dst, "data-extract")); err != nil {
			return err
		}
	}
	return nil
}

func extractArchive(casRoot, digest, dst string) error {
	src := casPath(casRoot, digest)
	os.MkdirAll(dst, 0o755)
	zstd := exec.Command("zstd", "-q", "-d", "-c", src)
	tar := exec.Command("tar", "-C", dst, "-xf", "-")
	zstd.Stdout, _ = tar.StdinPipe()
	if err := zstd.Start(); err != nil {
		return err
	}
	if err := tar.Start(); err != nil {
		return err
	}
	if err := zstd.Wait(); err != nil {
		return err
	}
	if err := tar.Wait(); err != nil {
		return err
	}
	// never resurrect gVisor filestore metadata from an old snapshot
	entries, _ := os.ReadDir(dst)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".gvisor.") {
			os.RemoveAll(filepath.Join(dst, e.Name()))
		}
	}
	return nil
}

// SnapshotManifest is stored alongside CAS for browsable snapshots.
type SnapshotManifest struct {
	RootDigest string `json:"rootDigest"`
	DataDigest string `json:"dataDigest"`
}

// WriteManifest / ReadManifest store snapshot CAS references in a sidecar dir.
func (m *Manager) WriteManifest(snapshotID, rootDigest, dataDigest string) error {
	dir := filepath.Join(m.BaseDir, "snapshots", snapshotID)
	os.MkdirAll(dir, 0o755)
	b, _ := json.Marshal(SnapshotManifest{RootDigest: rootDigest, DataDigest: dataDigest})
	return os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0o644)
}

func (m *Manager) ReadManifest(snapshotID string) (*SnapshotManifest, error) {
	b, err := os.ReadFile(filepath.Join(m.BaseDir, "snapshots", snapshotID, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var mf SnapshotManifest
	if json.Unmarshal(b, &mf) != nil {
		return nil, fmt.Errorf("bad manifest")
	}
	return &mf, nil
}

// ListSnapshotDir lists files of a snapshot by extracting to a temp dir.
func (m *Manager) ListSnapshotDir(snapshotID, guestPath string) ([]model.FileEntry, error) {
	mf, err := m.ReadManifest(snapshotID)
	if err != nil {
		return nil, err
	}
	tmp := filepath.Join(m.BaseDir, "tmp", "snapview-"+snapshotID)
	defer os.RemoveAll(tmp)
	if err := m.ExtractSnapshot(mf.RootDigest, "", tmp); err != nil {
		return nil, err
	}
	sb := &Sandbox{RootDir: filepath.Join(tmp, "root-extract"), DataDir: filepath.Join(tmp, "data-extract")}
	if _, err := os.Stat(sb.DataDir); err != nil {
		sb.DataDir = filepath.Join(tmp, "root-extract", "data-placeholder")
		os.MkdirAll(sb.DataDir, 0o755)
	}
	return sb.List(guestPath)
}

// sha256file for template digests
func fileDigest(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
