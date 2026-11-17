package api

import (
	"sync"

	"github.com/ibox/ibox/internal/model"
)

// TemplateRegistry holds template metadata. The actual rootfs content lives
// on the execution node as content-addressed CAS.
type TemplateRegistry struct {
	mu        sync.RWMutex
	byID      map[string][]model.Template // id -> versions
	latest    map[string]model.Template
	digestDir map[string]string // digest -> CAS path on node
}

func NewTemplateRegistry() *TemplateRegistry {
	tr := &TemplateRegistry{
		byID:      map[string][]model.Template{},
		latest:    map[string]model.Template{},
		digestDir: map[string]string{},
	}
	tr.seed()
	return tr
}

func (t *TemplateRegistry) seed() {
	standard := model.Template{
		ID: "standard", Version: "2026.11.1", Digest: "sha256:3fa8f928f1a3a9b2c0d4e5f60718293a4b5c6d7e8f90123456789abcdef01234",
		Description: "Debian 12 with build tools: python3, node 22, git, curl, jq, gcc, docker-cli. Home at /data/home.",
		Arch:        "aarch64",
		DefaultSpec: model.Spec{VCPUs: 4, MemMiB: 8192, PersistentDiskGiB: 20, RootDiskGiB: 16},
		Capabilities: []string{"identity-reset", "root-grow"},
		Creatable:   true, Latest: true,
	}
	minimal := model.Template{
		ID: "minimal", Version: "2026.11.1", Digest: "sha256:8e1f2a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f7",
		Description: "Debian 12 minbase: shell, coreutils, apt. Smallest start.",
		Arch:        "aarch64",
		DefaultSpec: model.Spec{VCPUs: 2, MemMiB: 4096, PersistentDiskGiB: 20, RootDiskGiB: 8},
		Capabilities: []string{"identity-reset", "root-grow"},
		Creatable:   true, Latest: true,
	}
	ci := model.Template{
		ID: "ci", Version: "2026.11.1", Digest: "sha256:1b2c3d4e5f60718293a4b5c6d7e8f9012a3b4c5d6e7f8091a2b3c4d5e6f70819",
		Description: "CI runner image: build tools + common runtimes for GitHub Actions style jobs.",
		Arch:        "aarch64",
		DefaultSpec: model.Spec{VCPUs: 4, MemMiB: 8192, PersistentDiskGiB: 20, RootDiskGiB: 16},
		Capabilities: []string{"identity-reset", "root-grow", "activity"},
		Creatable:   true, Latest: true,
	}
	// Pro-locked entries shown in the UI like the original product
	omarchy := model.Template{
		ID: "omarchy", Version: "2026.11.1", Digest: "sha256:5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c",
		Description: "Arch Linux with the Omarchy desktop (Pro)",
		Arch:        "aarch64",
		DefaultSpec: model.Spec{VCPUs: 4, MemMiB: 8192, PersistentDiskGiB: 20, RootDiskGiB: 16},
		Capabilities: []string{"identity-reset", "root-grow"},
		Hidden: true, Creatable: false, Latest: true,
	}
	android := model.Template{
		ID: "android", Version: "2026.11.1", Digest: "sha256:91a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5d6e7f8091a2b3c4d5e6f708192a3",
		Description: "Android 14 emulator with adb (Pro)",
		Arch:        "aarch64", Hidden: true, Creatable: false, Latest: true,
		DefaultSpec: model.Spec{VCPUs: 4, MemMiB: 8192, PersistentDiskGiB: 20, RootDiskGiB: 16},
	}
	gpu := model.Template{
		ID: "gpu", Version: "2026.11.1", Digest: "sha256:a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6",
		Description: "GPU (Linux) with CUDA toolkit (Pro)",
		Arch:        "aarch64", Hidden: true, Creatable: false, Latest: true,
		DefaultSpec: model.Spec{VCPUs: 11, MemMiB: 36864, PersistentDiskGiB: 20, RootDiskGiB: 16},
	}
	for _, tmpl := range []model.Template{standard, minimal, ci, omarchy, android, gpu} {
		t.byID[tmpl.ID] = append(t.byID[tmpl.ID], tmpl)
		if tmpl.Latest {
			t.latest[tmpl.ID] = tmpl
		}
	}
}

func (t *TemplateRegistry) List() []model.Template {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var out []model.Template
	for _, versions := range t.byID {
		out = append(out, versions...)
	}
	return out
}

func (t *TemplateRegistry) GetByID(id string) model.Template {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if l, ok := t.latest[id]; ok {
		return l
	}
	return model.Template{ID: id, Version: "unknown", Creatable: false}
}

// Resolve resolves "id" or "id@version".
func (t *TemplateRegistry) Resolve(id, version string) (model.Template, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if versions, ok := t.byID[id]; ok {
		if version == "" {
			if l, ok := t.latest[id]; ok {
				return l, true
			}
		}
		for _, v := range versions {
			if v.Version == version {
				return v, true
			}
		}
	}
	return model.Template{}, false
}
