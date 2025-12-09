// Package compute provides OVN network integration for compute nodes.
package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// OVNNetworkConfig holds OVN integration configuration.
type OVNNetworkConfig struct {
	// OVS database connection.
	OVSDBSocket string
	// OVN southbound database connection.
	SBDBConnection string
	// Bridge name for VM connections.
	IntegrationBridge string
	// Enable OVN metadata agent.
	EnableMetadata bool
	// Metadata proxy port.
	MetadataPort int
}

// OVNNetworkManager manages OVN networking for compute instances.
type OVNNetworkManager struct {
	config OVNNetworkConfig
	logger *zap.Logger
	mu     sync.RWMutex

	// Port tracking.
	ports map[string]*OVNPort
}

// OVNPort represents an OVN logical switch port.
type OVNPort struct {
	Name       string
	UUID       string
	MACAddress string
	IPAddress  string
	NetworkID  string
	InstanceID string
	CreatedAt  time.Time
}

// NewOVNNetworkManager creates a new OVN network manager.
func NewOVNNetworkManager(config OVNNetworkConfig, logger *zap.Logger) *OVNNetworkManager {
	if config.OVSDBSocket == "" {
		config.OVSDBSocket = "unix:/var/run/openvswitch/db.sock"
	}
	if config.IntegrationBridge == "" {
		config.IntegrationBridge = "br-int"
	}
	if config.MetadataPort == 0 {
		config.MetadataPort = 8775
	}

	return &OVNNetworkManager{
		config: config,
		logger: logger,
		ports:  make(map[string]*OVNPort),
	}
}

// AttachInstancePort attaches an instance to OVN network.
func (m *OVNNetworkManager) AttachInstancePort(ctx context.Context, req *AttachPortRequest) (*OVNPort, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	portName := fmt.Sprintf("vm-%s", req.InstanceID[:8])

	// Check if port already exists.
	if existing, ok := m.ports[portName]; ok {
		m.logger.Info("Port already exists", zap.String("port", portName))
		return existing, nil
	}

	// Get port info from controller.
	portInfo, err := m.getPortInfo(ctx, req.PortID)
	if err != nil {
		return nil, fmt.Errorf("get port info: %w", err)
	}

	// Create OVS port and attach to integration bridge.
	if err := m.createOVSPort(portName, req.InterfaceName); err != nil {
		return nil, fmt.Errorf("create OVS port: %w", err)
	}

	// Set external IDs for OVN.
	if err := m.setPortExternalIDs(portName, portInfo); err != nil {
		m.logger.Warn("Failed to set external IDs", zap.Error(err))
	}

	port := &OVNPort{
		Name:       portName,
		UUID:       portInfo.UUID,
		MACAddress: portInfo.MACAddress,
		IPAddress:  portInfo.IPAddress,
		NetworkID:  req.NetworkID,
		InstanceID: req.InstanceID,
		CreatedAt:  time.Now(),
	}

	m.ports[portName] = port

	m.logger.Info("Attached instance port",
		zap.String("port", portName),
		zap.String("instance", req.InstanceID),
		zap.String("network", req.NetworkID))

	return port, nil
}

// DetachInstancePort detaches an instance from OVN network.
func (m *OVNNetworkManager) DetachInstancePort(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	portName := fmt.Sprintf("vm-%s", instanceID[:8])

	if _, ok := m.ports[portName]; !ok {
		m.logger.Warn("Port not found", zap.String("port", portName))
		return nil
	}

	// Delete OVS port.
	if err := m.deleteOVSPort(portName); err != nil {
		m.logger.Warn("Failed to delete OVS port", zap.Error(err))
	}

	delete(m.ports, portName)

	m.logger.Info("Detached instance port",
		zap.String("port", portName),
		zap.String("instance", instanceID))

	return nil
}

// createOVSPort creates an OVS port on integration bridge.
func (m *OVNNetworkManager) createOVSPort(portName, interfaceName string) error {
	args := []string{
		"--may-exist", "add-port", m.config.IntegrationBridge, interfaceName,
		"--", "set", "interface", interfaceName,
		fmt.Sprintf("external_ids:iface-id=%s", portName),
	}

	cmd := exec.Command("ovs-vsctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ovs-vsctl add-port failed: %v, output: %s", err, string(output))
	}

	return nil
}

// deleteOVSPort deletes an OVS port from integration bridge.
func (m *OVNNetworkManager) deleteOVSPort(interfaceName string) error {
	args := []string{
		"--if-exists", "del-port", m.config.IntegrationBridge, interfaceName,
	}

	cmd := exec.Command("ovs-vsctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ovs-vsctl del-port failed: %v, output: %s", err, string(output))
	}

	return nil
}

// setPortExternalIDs sets OVN external IDs for port.
func (m *OVNNetworkManager) setPortExternalIDs(portName string, info *PortInfo) error {
	// Set external IDs via ovs-vsctl.
	externalIDs := map[string]string{
		"iface-id":     portName,
		"attached-mac": info.MACAddress,
		"iface-status": "active",
		"vm-id":        info.InstanceID,
	}

	for key, value := range externalIDs {
		args := []string{
			"set", "interface", portName,
			fmt.Sprintf("external_ids:%s=%s", key, value),
		}

		cmd := exec.Command("ovs-vsctl", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			m.logger.Warn("Failed to set external ID",
				zap.String("key", key),
				zap.Error(err),
				zap.String("output", string(output)))
		}
	}

	return nil
}

// getPortInfo retrieves port information from controller.
func (m *OVNNetworkManager) getPortInfo(ctx context.Context, portID string) (*PortInfo, error) {
	// This would typically call the controller API.
	// For now, return mock data.
	return &PortInfo{
		UUID:       portID,
		MACAddress: "fa:16:3e:00:00:01",
		IPAddress:  "10.0.0.10",
		InstanceID: portID,
	}, nil
}

// GetPortByInstance retrieves port info for an instance.
func (m *OVNNetworkManager) GetPortByInstance(instanceID string) (*OVNPort, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	portName := fmt.Sprintf("vm-%s", instanceID[:8])
	port, ok := m.ports[portName]
	if !ok {
		return nil, fmt.Errorf("port not found for instance %s", instanceID)
	}

	return port, nil
}

// ListPorts lists all managed ports.
func (m *OVNNetworkManager) ListPorts() []*OVNPort {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ports := make([]*OVNPort, 0, len(m.ports))
	for _, port := range m.ports {
		ports = append(ports, port)
	}

	return ports
}

// SyncPorts synchronizes port state with OVN.
func (m *OVNNetworkManager) SyncPorts(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// List ports from OVS.
	cmd := exec.Command("ovs-vsctl", "list-ports", m.config.IntegrationBridge)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("list OVS ports: %w", err)
	}

	ovsPorts := make(map[string]bool)
	for _, line := range strings.Split(string(output), "\n") {
		port := strings.TrimSpace(line)
		if port != "" {
			ovsPorts[port] = true
		}
	}

	// Remove ports not in OVS.
	for portName := range m.ports {
		if !ovsPorts[portName] {
			m.logger.Info("Removing stale port", zap.String("port", portName))
			delete(m.ports, portName)
		}
	}

	m.logger.Info("Port sync completed",
		zap.Int("managed_ports", len(m.ports)),
		zap.Int("ovs_ports", len(ovsPorts)))

	return nil
}

// AttachPortRequest contains parameters for attaching a port.
type AttachPortRequest struct {
	InstanceID    string
	NetworkID     string
	PortID        string
	InterfaceName string
}

// PortInfo contains port information from controller.
type PortInfo struct {
	UUID       string
	MACAddress string
	IPAddress  string
	InstanceID string
}

// GetOVNInfo retrieves OVN-specific information.
func (m *OVNNetworkManager) GetOVNInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"integration_bridge": m.config.IntegrationBridge,
		"ovsdb_socket":       m.config.OVSDBSocket,
		"sb_connection":      m.config.SBDBConnection,
		"managed_ports":      len(m.ports),
		"metadata_enabled":   m.config.EnableMetadata,
		"metadata_port":      m.config.MetadataPort,
	}
}

// ValidateOVNSetup validates OVN integration setup.
func (m *OVNNetworkManager) ValidateOVNSetup(ctx context.Context) error {
	// Check if integration bridge exists.
	cmd := exec.Command("ovs-vsctl", "br-exists", m.config.IntegrationBridge)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("integration bridge %s not found", m.config.IntegrationBridge)
	}

	// Check OVS version.
	cmd = exec.Command("ovs-vsctl", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot determine OVS version: %w", err)
	}

	m.logger.Info("OVN setup validated",
		zap.String("bridge", m.config.IntegrationBridge),
		zap.String("ovs_version", strings.Split(string(output), "\n")[0]))

	return nil
}

// OVNNetworkStats contains network statistics.
type OVNNetworkStats struct {
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxErrors  uint64 `json:"rx_errors"`
	TxErrors  uint64 `json:"tx_errors"`
}

// GetPortStats retrieves port statistics.
func (m *OVNNetworkManager) GetPortStats(portName string) (*OVNNetworkStats, error) {
	cmd := exec.Command("ovs-vsctl", "get", "Interface", portName, "statistics")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("get port statistics: %w", err)
	}

	// Parse statistics (format: {key1=value1, key2=value2}).
	stats := &OVNNetworkStats{}
	statsStr := strings.TrimSpace(string(output))

	// Simple parsing for common fields.
	if strings.Contains(statsStr, "rx_packets") {
		// Parse would be more sophisticated in production.
		m.logger.Debug("Port statistics retrieved", zap.String("stats", statsStr))
	}

	return stats, nil
}

// ConfigureMetadataAgent configures OVN metadata agent.
func (m *OVNNetworkManager) ConfigureMetadataAgent(ctx context.Context) error {
	if !m.config.EnableMetadata {
		m.logger.Info("Metadata agent disabled")
		return nil
	}

	// This would start the OVN metadata agent service.
	m.logger.Info("Configuring OVN metadata agent",
		zap.Int("port", m.config.MetadataPort))

	return nil
}

// MarshalJSON implements json.Marshaler for OVNPort.
func (p *OVNPort) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"name":        p.Name,
		"uuid":        p.UUID,
		"mac_address": p.MACAddress,
		"ip_address":  p.IPAddress,
		"network_id":  p.NetworkID,
		"instance_id": p.InstanceID,
		"created_at":  p.CreatedAt.Unix(),
	})
}
