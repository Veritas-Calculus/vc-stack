// Package compute provides QEMU/KVM configuration management.
package compute

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QEMUConfig represents QEMU/KVM VM configuration.
type QEMUConfig struct {
	// VM identification.
	ID        string `json:"id"`
	Name      string `json:"name"`
	TenantID  string `json:"tenant_id"`
	ProjectID string `json:"project_id"`

	// Resources.
	VCPUs    int `json:"vcpus"`
	MemoryMB int `json:"memory_mb"`
	DiskGB   int `json:"disk_gb"`

	// Images.
	ImageID   string `json:"image_id"`
	ImagePath string `json:"image_path"`
	DiskPath  string `json:"disk_path"`

	// Networking.
	Networks []NetworkConfig `json:"networks"`

	// QEMU process.
	PID        int    `json:"pid"`
	SocketPath string `json:"socket_path"`
	VNCPort    int    `json:"vnc_port"`

	// Firmware and boot options.
	BootMode   string `json:"boot_mode"`   // bios or uefi
	EnableTPM  bool   `json:"enable_tpm"`  // Enable TPM 2.0
	SecureBoot bool   `json:"secure_boot"` // Enable secure boot (requires UEFI)

	// Metadata.
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	StartedAt time.Time `json:"started_at,omitempty"`

	// Cloud-init.
	UserData string `json:"user_data,omitempty"`
	MetaData string `json:"meta_data,omitempty"`
}

// NetworkConfig represents VM network configuration.
type NetworkConfig struct {
	NetworkID string `json:"network_id"`
	PortID    string `json:"port_id"`
	MAC       string `json:"mac"`
	IP        string `json:"ip"`
	Interface string `json:"interface"`
}

// QEMUConfigStore manages VM configurations on disk.
type QEMUConfigStore struct {
	baseDir string
	logger  *zap.Logger
	mu      sync.RWMutex
	cache   map[string]*QEMUConfig
}

// NewQEMUConfigStore creates a new configuration store.
func NewQEMUConfigStore(baseDir string, logger *zap.Logger) (*QEMUConfigStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config directory: %w", err)
	}

	store := &QEMUConfigStore{
		baseDir: baseDir,
		logger:  logger,
		cache:   make(map[string]*QEMUConfig),
	}

	// Load existing configurations.
	if err := store.loadAll(); err != nil {
		logger.Warn("Failed to load existing configs", zap.Error(err))
	}

	return store, nil
}

// Save saves VM configuration to disk.
func (s *QEMUConfigStore) Save(config *QEMUConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	configPath := s.getConfigPath(config.ID)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	s.cache[config.ID] = config

	s.logger.Debug("Saved VM config",
		zap.String("id", config.ID),
		zap.String("name", config.Name),
		zap.String("status", config.Status))

	return nil
}

// Load loads VM configuration from disk.
func (s *QEMUConfigStore) Load(id string) (*QEMUConfig, error) {
	s.mu.RLock()
	if cached, ok := s.cache[id]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	configPath := s.getConfigPath(id)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("VM config not found: %s", id)
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config QEMUConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	s.mu.Lock()
	s.cache[id] = &config
	s.mu.Unlock()

	return &config, nil
}

// Delete deletes VM configuration from disk.
func (s *QEMUConfigStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	configPath := s.getConfigPath(id)
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete config file: %w", err)
	}

	delete(s.cache, id)

	s.logger.Debug("Deleted VM config", zap.String("id", id))
	return nil
}

// List lists all VM configurations.
func (s *QEMUConfigStore) List() ([]*QEMUConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs := make([]*QEMUConfig, 0, len(s.cache))
	for _, config := range s.cache {
		configs = append(configs, config)
	}

	return configs, nil
}

// ListByStatus lists VMs by status.
func (s *QEMUConfigStore) ListByStatus(status string) ([]*QEMUConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs := make([]*QEMUConfig, 0)
	for _, config := range s.cache {
		if config.Status == status {
			configs = append(configs, config)
		}
	}

	return configs, nil
}

// UpdateStatus updates VM status.
func (s *QEMUConfigStore) UpdateStatus(id, status string) error {
	config, err := s.Load(id)
	if err != nil {
		return err
	}

	config.Status = status
	if status == "running" && config.StartedAt.IsZero() {
		config.StartedAt = time.Now()
	}

	return s.Save(config)
}

// UpdatePID updates VM process ID.
func (s *QEMUConfigStore) UpdatePID(id string, pid int) error {
	config, err := s.Load(id)
	if err != nil {
		return err
	}

	config.PID = pid
	return s.Save(config)
}

// getConfigPath returns path to configuration file.
func (s *QEMUConfigStore) getConfigPath(id string) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%s.json", id))
}

// loadAll loads all existing configurations.
func (s *QEMUConfigStore) loadAll() error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("read config directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		id := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		if _, err := s.Load(id); err != nil {
			s.logger.Warn("Failed to load config",
				zap.String("file", entry.Name()),
				zap.Error(err))
		}
	}

	s.logger.Info("Loaded VM configs",
		zap.Int("count", len(s.cache)))

	return nil
}

// Exists checks if configuration exists.
func (s *QEMUConfigStore) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.cache[id]
	return ok
}

// GetByName retrieves configuration by name.
func (s *QEMUConfigStore) GetByName(name string) (*QEMUConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, config := range s.cache {
		if config.Name == name {
			return config, nil
		}
	}

	return nil, fmt.Errorf("VM not found: %s", name)
}

// Count returns total number of configurations.
func (s *QEMUConfigStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.cache)
}

// CountByStatus returns number of VMs in given status.
func (s *QEMUConfigStore) CountByStatus(status string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, config := range s.cache {
		if config.Status == status {
			count++
		}
	}
	return count
}
