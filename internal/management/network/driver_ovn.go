package network

import (
	"fmt"
	"os/exec"

	"go.uber.org/zap"
)

// OVNDriver implements Driver using OVN NBDB.
type OVNDriver struct {
	logger *zap.Logger
	dbAddr string // e.g. tcp:127.0.0.1:6641
}

func NewOVNDriver(logger *zap.Logger, cfg OVNConfig) *OVNDriver {
	return &OVNDriver{logger: logger, dbAddr: cfg.NBAddress}
}

func (d *OVNDriver) nbctl(args ...string) error {
	fullArgs := append([]string{"--db=" + d.dbAddr}, args...)

	// #nosec G204 - OVN commands are internally generated and safe.
	cmd := exec.Command("ovn-nbctl", fullArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ovn-nbctl failed: %w (output: %s)", err, string(out))
	}
	return nil
}

func (d *OVNDriver) EnsureNetwork(n *Network, s *Subnet) error {
	_ = d.nbctl("--may-exist", "ls-add", n.ID)
	if s != nil {
		_ = d.nbctl("set", "Logical_Switch", n.ID, "other_config:subnet="+s.CIDR)
	}
	return nil
}

func (d *OVNDriver) DeleteNetwork(n *Network) error {
	return d.nbctl("ls-del", n.ID)
}

func (d *OVNDriver) EnsurePort(n *Network, s *Subnet, p *NetworkPort) error {
	_ = d.nbctl("--may-exist", "lsp-add", n.ID, p.ID)
	_ = d.nbctl("lsp-set-addresses", p.ID, p.MACAddress+" "+p.FixedIPs[0].IP)
	return nil
}

func (d *OVNDriver) DeletePort(n *Network, p *NetworkPort) error {
	return d.nbctl("lsp-del", p.ID)
}

func (d *OVNDriver) EnsureRouter(name string) error {
	return d.nbctl("--may-exist", "lr-add", name)
}

func (d *OVNDriver) DeleteRouter(name string) error {
	return d.nbctl("lr-del", name)
}

func (d *OVNDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	portName := fmt.Sprintf("rp-%s-%s", router, n.ID)
	_ = d.nbctl("--may-exist", "lrp-add", router, portName, GenerateMAC(), s.Gateway)
	_ = d.nbctl("--may-exist", "lsp-add", n.ID, "lp-"+portName)
	_ = d.nbctl("lsp-set-type", "lp-"+portName, "router")
	_ = d.nbctl("lsp-set-addresses", "lp-"+portName, "router")
	_ = d.nbctl("lsp-set-options", "lp-"+portName, "router-port="+portName)
	return nil
}

func (d *OVNDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	portName := fmt.Sprintf("rp-%s-%s", router, n.ID)
	_ = d.nbctl("lrp-del", portName)
	_ = d.nbctl("lsp-del", "lp-"+portName)
	return nil
}

func (d *OVNDriver) SetRouterGateway(router string, extNet *Network, extSub *Subnet) (string, error) {
	return "", nil
}

func (d *OVNDriver) ClearRouterGateway(router string, extNet *Network) error {
	return nil
}

func (d *OVNDriver) SetRouterSNAT(router string, enable bool, internalCIDR, externalIP string) error {
	if enable {
		return d.nbctl("lr-nat-add", router, "snat", externalIP, internalCIDR)
	}
	return d.nbctl("lr-nat-del", router, "snat", internalCIDR)
}

func (d *OVNDriver) EnsureFIPNAT(router, floatingIP, fixedIP string) error {
	return d.nbctl("lr-nat-add", router, "dnat_and_snat", floatingIP, fixedIP)
}

func (d *OVNDriver) RemoveFIPNAT(router, floatingIP, fixedIP string) error {
	return d.nbctl("lr-nat-del", router, "dnat_and_snat", floatingIP)
}

// --- Port Group / Security Group Implementation ---

func (d *OVNDriver) EnsurePortGroup(name string) error {
	pgName := "pg-" + name
	return d.nbctl("--may-exist", "pg-add", pgName)
}

func (d *OVNDriver) DeletePortGroup(name string) error {
	return d.nbctl("pg-del", "pg-"+name)
}

func (d *OVNDriver) SetPortGroupACLs(name string, rules []ACLRule) error {
	pgName := "pg-" + name
	_ = d.nbctl("pg-acl-del", pgName)
	for _, r := range rules {
		logStr := "false"
		if r.Log { logStr = "true" }
		_ = d.nbctl("acl-add", pgName, r.Direction, fmt.Sprintf("%d", r.Priority), r.Match, r.Action, "--log="+logStr)
	}
	return nil
}

func (d *OVNDriver) AddPortToPortGroup(portID, groupName string) error {
	return d.nbctl("pg-set-ports", "pg-"+groupName, portID)
}

func (d *OVNDriver) RemovePortFromPortGroup(portID, groupName string) error {
	return nil
}

func (d *OVNDriver) SetPortTag(portID string, parentPortID string, tag int) error {
	// #nosec G204
	return d.nbctl("set", "Logical_Switch_Port", portID, fmt.Sprintf("tag=%d", tag))
}

func (d *OVNDriver) ClearPortTag(portID string) error {
	return d.nbctl("clear", "Logical_Switch_Port", portID, "tag")
}

func (d *OVNDriver) AddStaticRoute(routerName, destination, nexthop string) error {
	return d.nbctl("lr-route-add", routerName, destination, nexthop)
}

func (d *OVNDriver) DeleteStaticRoute(routerName, destination, nexthop string) error {
	return d.nbctl("lr-route-del", routerName, destination)
}

