// Package compute provides compute service for the VC Stack controller.
// This file defines the PortAllocator interface that decouples compute
// from network implementation details.
package compute

// PortAllocator abstracts network port allocation for VM creation.
// The network service implements this interface, and the compute service
// uses it to allocate/deallocate ports during instance lifecycle operations.
type PortAllocator interface {
	// AllocatePort creates a network port with IP allocation and OVN LSP provisioning.
	// It returns the MAC address, port ID, and allocated fixed IP.
	// If requestedIP is non-empty, that specific IP is used instead of IPAM allocation.
	// If securityGroupIDs is empty, the default security group for the tenant is applied.
	AllocatePort(networkID, deviceID, tenantID, requestedIP string, securityGroupIDs []string) (
		mac string, portID string, fixedIP string, err error)

	// DeallocatePort releases a port and its associated IPAM allocation (on VM delete).
	DeallocatePort(portID string) error

	// DefaultNetworkID returns the first non-external network for a tenant,
	// or empty string if none exists.
	DefaultNetworkID(tenantID string) (string, error)

	// GetPortIP returns the primary fixed IP for a port.
	GetPortIP(portID string) string
}
