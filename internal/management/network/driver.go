package network


// Driver abstracts SDN backend operations used by network service.
type Driver interface {
	EnsureNetwork(n *Network, s *Subnet) error
	DeleteNetwork(n *Network) error
	EnsurePort(n *Network, s *Subnet, p *NetworkPort) error
	DeletePort(n *Network, p *NetworkPort) error
	EnsureRouter(name string) error
	DeleteRouter(name string) error
	ConnectSubnetToRouter(router string, n *Network, s *Subnet) error
	DisconnectSubnetFromRouter(router string, n *Network) error
	SetRouterGateway(router string, extNet *Network, extSub *Subnet) (string, error)
	ClearRouterGateway(router string, extNet *Network) error
	SetRouterSNAT(router string, enable bool, internalCIDR, externalIP string) error
	EnsureFIPNAT(router, floatingIP, fixedIP string) error
	RemoveFIPNAT(router, floatingIP, fixedIP string) error
	
	// Security Groups via Port Groups
	EnsurePortGroup(name string) error
	DeletePortGroup(name string) error
	SetPortGroupACLs(name string, rules []ACLRule) error
	AddPortToPortGroup(portID, groupName string) error
	RemovePortFromPortGroup(portID, groupName string) error

	// Advanced features
	SetPortTag(portID string, parentPortID string, tag int) error
	ClearPortTag(portID string) error
	AddStaticRoute(routerName, destination, nexthop string) error
	DeleteStaticRoute(routerName, destination, nexthop string) error
}

// ACLRule represents a single ACL entry.
type ACLRule struct {
	Priority  int
	Direction string // "to-lport" (ingress) or "from-lport" (egress)
	Match     string // OVN match syntax
	Action    string // allow, allow-related, drop, reject
	Log       bool
}

// CompiledSecurityGroup is used for legacy port security.
type CompiledSecurityGroup struct {
	Name  string
	Rules []ACLRule
}
