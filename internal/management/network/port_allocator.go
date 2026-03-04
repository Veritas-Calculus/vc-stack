package network

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// AllocatePort implements compute.PortAllocator.
// It creates a network port with IP allocation, OVN LSP provisioning,
// and default security group assignment.
func (s *Service) AllocatePort(networkID, deviceID, tenantID, requestedIP string, securityGroupIDs []string) (
	mac string, portID string, fixedIP string, err error) {
	// 1. Find network.
	var network Network
	if err := s.db.First(&network, "id = ?", networkID).Error; err != nil {
		return "", "", "", fmt.Errorf("network not found: %s: %w", networkID, err)
	}

	// 2. Find first subnet for this network.
	var subnet Subnet
	if err := s.db.Where("network_id = ?", network.ID).First(&subnet).Error; err != nil {
		return "", "", "", fmt.Errorf("no subnet found for network %s: %w", network.ID, err)
	}

	// 3. Allocate IP via IPAM (or use requested IP).
	ip := requestedIP
	if ip == "" {
		allocated, err := s.ipam.Allocate(&subnet, "")
		if err != nil {
			return "", "", "", fmt.Errorf("IP allocation failed for subnet %s: %w", subnet.ID, err)
		}
		ip = allocated
	}

	// 4. Generate MAC address.
	macAddr := GenerateMAC()

	// 5. Determine security groups.
	sgStr := ""
	if len(securityGroupIDs) > 0 {
		sgStr = strings.Join(securityGroupIDs, ",")
	} else {
		// Auto-assign default security group.
		defaultSG, sgErr := s.EnsureDefaultSecurityGroup(tenantID)
		if sgErr != nil {
			s.logger.Warn("failed to ensure default security group", zap.Error(sgErr))
		} else if defaultSG != nil {
			sgStr = defaultSG.ID
		}
	}

	// 6. Create port record.
	port := NetworkPort{
		ID:             generateUUID(),
		NetworkID:      network.ID,
		SubnetID:       subnet.ID,
		MACAddress:     macAddr,
		FixedIPs:       FixedIPList{{IP: ip, SubnetID: subnet.ID}},
		DeviceID:       deviceID,
		DeviceOwner:    "compute:instance",
		SecurityGroups: sgStr,
		Status:         "building",
		TenantID:       tenantID,
	}
	if err := s.db.Create(&port).Error; err != nil {
		return "", "", "", fmt.Errorf("failed to create port: %w", err)
	}

	// 7. Update IPAM allocation with port ID.
	s.db.Model(&IPAllocation{}).Where("subnet_id = ? AND ip = ?", subnet.ID, ip).
		Update("port_id", port.ID)

	// 8. OVN: create LSP + DHCP binding.
	if err := s.driver.EnsurePort(&network, &subnet, &port); err != nil {
		s.logger.Error("EnsurePort failed (port created in DB, OVN may need repair)",
			zap.Error(err), zap.String("port_id", port.ID))
	}

	// 9. Apply security group ACLs to OVN.
	if sgStr != "" {
		if err := s.applyPortSecurityACLs(&port); err != nil {
			s.logger.Warn("failed to apply SG ACLs to port",
				zap.Error(err), zap.String("port_id", port.ID))
		}
	}

	// 10. Update port status.
	s.db.Model(&port).Update("status", "active")

	s.logger.Info("port allocated",
		zap.String("port_id", port.ID),
		zap.String("mac", macAddr),
		zap.String("ip", ip),
		zap.String("network_id", networkID),
		zap.String("device_id", deviceID))

	return macAddr, port.ID, ip, nil
}

// DeallocatePort implements compute.PortAllocator.
// It removes the port from OVN and releases the IP allocation.
func (s *Service) DeallocatePort(portID string) error {
	var port NetworkPort
	if err := s.db.First(&port, "id = ?", portID).Error; err != nil {
		return fmt.Errorf("port not found: %w", err)
	}

	var network Network
	if err := s.db.First(&network, "id = ?", port.NetworkID).Error; err != nil {
		s.logger.Warn("network not found for port cleanup", zap.String("port_id", portID))
	} else {
		// Remove from OVN.
		if err := s.driver.DeletePort(&network, &port); err != nil {
			s.logger.Warn("OVN port cleanup failed", zap.Error(err), zap.String("port_id", portID))
		}
	}

	// Release IP allocations.
	s.db.Where("port_id = ?", portID).Delete(&IPAllocation{})

	// Delete port record.
	if err := s.db.Delete(&port).Error; err != nil {
		return fmt.Errorf("failed to delete port: %w", err)
	}

	s.logger.Info("port deallocated", zap.String("port_id", portID))
	return nil
}

// DefaultNetworkID implements compute.PortAllocator.
// Returns the first non-external network for a tenant, or empty if none exists.
func (s *Service) DefaultNetworkID(tenantID string) (string, error) {
	var network Network
	err := s.db.Where("tenant_id = ? AND external = false AND status = 'active'", tenantID).
		Order("created_at asc").First(&network).Error
	if err != nil {
		// Try shared networks.
		err = s.db.Where("shared = true AND external = false AND status = 'active'").
			Order("created_at asc").First(&network).Error
		if err != nil {
			return "", nil // No default network, not an error.
		}
	}
	return network.ID, nil
}

// GetPortIP implements compute.PortAllocator.
// Returns the primary fixed IP for a port.
func (s *Service) GetPortIP(portID string) string {
	var port NetworkPort
	if err := s.db.First(&port, "id = ?", portID).Error; err != nil {
		return ""
	}
	if len(port.FixedIPs) > 0 {
		return port.FixedIPs[0].IP
	}
	return ""
}
