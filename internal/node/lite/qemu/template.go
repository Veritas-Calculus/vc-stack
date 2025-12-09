package qemu

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Template represents a VM configuration template.
type Template struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	Config      VMConfig          `json:"config"`
	CreatedAt   int64             `json:"created_at"`
	UpdatedAt   int64             `json:"updated_at"`
}

// TemplateStore manages VM configuration templates.
type TemplateStore struct {
	basePath string
}

// NewTemplateStore creates a new template store.
func NewTemplateStore(basePath string) (*TemplateStore, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("create template dir: %w", err)
	}
	return &TemplateStore{basePath: basePath}, nil
}

// Save saves a template.
func (s *TemplateStore) Save(tmpl *Template) error {
	path := filepath.Join(s.basePath, tmpl.Name+".json")
	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal template: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write template: %w", err)
	}
	return nil
}

// Load loads a template by name.
func (s *TemplateStore) Load(name string) (*Template, error) {
	path := filepath.Join(s.basePath, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	var tmpl Template
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, fmt.Errorf("unmarshal template: %w", err)
	}
	return &tmpl, nil
}

// List lists all available templates.
func (s *TemplateStore) List() ([]*Template, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("read template dir: %w", err)
	}

	templates := []*Template{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		name := entry.Name()[:len(entry.Name())-5] // remove .json
		tmpl, err := s.Load(name)
		if err != nil {
			continue
		}
		templates = append(templates, tmpl)
	}
	return templates, nil
}

// Delete deletes a template.
func (s *TemplateStore) Delete(name string) error {
	path := filepath.Join(s.basePath, name+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	return nil
}

// DefaultTemplates returns a set of default templates.
func DefaultTemplates() []*Template {
	return []*Template{
		{
			Name:        "ubuntu-server",
			Description: "Ubuntu Server with cloud-init",
			Labels:      map[string]string{"os": "ubuntu", "type": "server"},
			Config: VMConfig{
				VCPUs:       2,
				MemoryMB:    2048,
				CPUModel:    "host",
				MachineType: "q35",
				Boot:        []string{"c"},
				Disks: []DiskConfig{
					{Type: "file", Format: "qcow2", Bus: "virtio", Cache: "none", AIO: "native", SizeGB: 20},
				},
				NICs: []NICConfig{
					{Type: "tap", Model: "virtio-net-pci"},
				},
				VNC: VNCConfig{Enabled: true, Listen: "0.0.0.0"},
				QMP: QMPConfig{Enabled: true, Type: "unix"},
			},
		},
		{
			Name:        "ubuntu-desktop",
			Description: "Ubuntu Desktop with UEFI and graphics",
			Labels:      map[string]string{"os": "ubuntu", "type": "desktop"},
			Config: VMConfig{
				VCPUs:       4,
				MemoryMB:    4096,
				CPUModel:    "host",
				MachineType: "q35",
				UEFI:        true,
				Boot:        []string{"c"},
				Disks: []DiskConfig{
					{Type: "file", Format: "qcow2", Bus: "virtio", Cache: "none", AIO: "native", SizeGB: 40},
				},
				NICs: []NICConfig{
					{Type: "tap", Model: "virtio-net-pci"},
				},
				VNC: VNCConfig{Enabled: true, Listen: "0.0.0.0"},
				QMP: QMPConfig{Enabled: true, Type: "unix"},
			},
		},
		{
			Name:        "windows-server",
			Description: "Windows Server with UEFI and TPM",
			Labels:      map[string]string{"os": "windows", "type": "server"},
			Config: VMConfig{
				VCPUs:       4,
				MemoryMB:    8192,
				CPUModel:    "host",
				MachineType: "q35",
				UEFI:        true,
				TPM:         true,
				Boot:        []string{"c"},
				Disks: []DiskConfig{
					{Type: "file", Format: "qcow2", Bus: "virtio", Cache: "none", AIO: "native", SizeGB: 60},
				},
				NICs: []NICConfig{
					{Type: "tap", Model: "virtio-net-pci"},
				},
				VNC: VNCConfig{Enabled: true, Listen: "0.0.0.0"},
				QMP: QMPConfig{Enabled: true, Type: "unix"},
			},
		},
		{
			Name:        "minimal",
			Description: "Minimal VM with 1 vCPU and 512MB RAM",
			Labels:      map[string]string{"type": "minimal"},
			Config: VMConfig{
				VCPUs:       1,
				MemoryMB:    512,
				CPUModel:    "host",
				MachineType: "pc",
				Boot:        []string{"c"},
				Disks: []DiskConfig{
					{Type: "file", Format: "qcow2", Bus: "virtio", Cache: "none", SizeGB: 10},
				},
				NICs: []NICConfig{
					{Type: "user", Model: "virtio-net-pci"},
				},
				VNC: VNCConfig{Enabled: true, Listen: "127.0.0.1"},
				QMP: QMPConfig{Enabled: true, Type: "unix"},
			},
		},
	}
}
