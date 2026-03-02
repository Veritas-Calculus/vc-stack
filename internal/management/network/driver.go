package network

import "go.uber.org/zap"

// Driver abstracts SDN backend operations used by network service.
type Driver interface {
	EnsureNetwork(n *Network, s *Subnet) error
	DeleteNetwork(n *Network) error
	EnsurePort(n *Network, s *Subnet, p *NetworkPort) error
	DeletePort(n *Network, p *NetworkPort) error
	// Router operations.
	EnsureRouter(name string) error
	DeleteRouter(name string) error
	// Connect a subnet's logical switch to a logical router (create LRP and router-type LSP)
	ConnectSubnetToRouter(router string, n *Network, s *Subnet) error
	DisconnectSubnetFromRouter(router string, n *Network) error
	// Set external gateway for router (connects router to external network)
	SetRouterGateway(router string, externalNetwork *Network, externalSubnet *Subnet) (gatewayIP string, err error)
	ClearRouterGateway(router string, externalNetwork *Network) error
	// Enable/disable SNAT on router for outbound traffic.
	SetRouterSNAT(router string, enable bool, internalCIDR string, externalIP string) error
	// NAT for Floating IPs on a logical router.
	EnsureFIPNAT(router string, floatingIP, fixedIP string) error
	RemoveFIPNAT(router string, floatingIP, fixedIP string) error
	// ACLs per port (compiled from security groups)
	ReplacePortACLs(networkID, portID string, rules []ACLRule) error
	// Security Groups via Port Groups: ensure PGs and ACLs, and port membership.
	EnsurePortSecurity(portID string, groups []CompiledSecurityGroup) error
}

// ACLRule represents a single ACL entry to apply on a port.
type ACLRule struct {
	Direction string // "to-lport" (ingress) or "from-lport" (egress)
	Priority  int
	Match     string
	Action    string // allow/allow-related/drop
}

// CompiledSecurityGroup contains SG id and compiled ACL rules for that SG.
type CompiledSecurityGroup struct {
	ID    string
	Rules []ACLRule
}

// NoopDriver is a stub driver that performs no SDN actions.
type NoopDriver struct{ logger *zap.Logger }

func NewNoopDriver(l *zap.Logger) *NoopDriver { return &NoopDriver{logger: l} }

func (d *NoopDriver) EnsureNetwork(n *Network, s *Subnet) error {
	d.logger.Debug("noop EnsureNetwork", zap.String("id", n.ID))
	return nil
}
func (d *NoopDriver) DeleteNetwork(n *Network) error {
	d.logger.Debug("noop DeleteNetwork", zap.String("id", n.ID))
	return nil
}
func (d *NoopDriver) EnsurePort(n *Network, s *Subnet, p *NetworkPort) error {
	d.logger.Debug("noop EnsurePort", zap.String("id", p.ID))
	return nil
}
func (d *NoopDriver) DeletePort(n *Network, p *NetworkPort) error {
	d.logger.Debug("noop DeletePort", zap.String("id", p.ID))
	return nil
}

func (d *NoopDriver) EnsureRouter(name string) error {
	d.logger.Debug("noop EnsureRouter", zap.String("name", name))
	return nil
}
func (d *NoopDriver) DeleteRouter(name string) error {
	d.logger.Debug("noop DeleteRouter", zap.String("name", name))
	return nil
}
func (d *NoopDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	d.logger.Debug("noop ConnectSubnetToRouter", zap.String("router", router), zap.String("network", n.ID), zap.String("subnet", s.ID))
	return nil
}
func (d *NoopDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	d.logger.Debug("noop DisconnectSubnetFromRouter", zap.String("router", router), zap.String("network", n.ID))
	return nil
}
func (d *NoopDriver) SetRouterGateway(router string, externalNetwork *Network, externalSubnet *Subnet) (string, error) {
	d.logger.Debug("noop SetRouterGateway", zap.String("router", router), zap.String("external_network", externalNetwork.ID))
	return "10.0.0.1", nil
}
func (d *NoopDriver) ClearRouterGateway(router string, externalNetwork *Network) error {
	d.logger.Debug("noop ClearRouterGateway", zap.String("router", router))
	return nil
}
func (d *NoopDriver) SetRouterSNAT(router string, enable bool, internalCIDR, externalIP string) error {
	d.logger.Debug("noop SetRouterSNAT", zap.String("router", router), zap.Bool("enable", enable))
	return nil
}
func (d *NoopDriver) EnsureFIPNAT(router, floatingIP, fixedIP string) error {
	d.logger.Debug("noop EnsureFIPNAT", zap.String("router", router), zap.String("fip", floatingIP), zap.String("fixed", fixedIP))
	return nil
}
func (d *NoopDriver) RemoveFIPNAT(router, floatingIP, fixedIP string) error {
	d.logger.Debug("noop RemoveFIPNAT", zap.String("router", router), zap.String("fip", floatingIP), zap.String("fixed", fixedIP))
	return nil
}
func (d *NoopDriver) ReplacePortACLs(networkID, portID string, rules []ACLRule) error {
	d.logger.Debug("noop ReplacePortACLs", zap.String("network", networkID), zap.String("port", portID))
	return nil
}

func (d *NoopDriver) EnsurePortSecurity(portID string, groups []CompiledSecurityGroup) error {
	d.logger.Debug("noop EnsurePortSecurity", zap.String("port", portID))
	return nil
}
