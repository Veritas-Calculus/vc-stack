package network

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// FallbackDriver tries primary driver first, and on failure falls back to secondary.
// This makes Network Service resilient when a plugin endpoint is configured but not available.
type FallbackDriver struct {
	logger    *zap.Logger
	primary   Driver
	secondary Driver
}

func NewFallbackDriver(l *zap.Logger, primary, secondary Driver) *FallbackDriver {
	if l == nil {
		l = zap.NewNop()
	}
	return &FallbackDriver{logger: l, primary: primary, secondary: secondary}
}

// try calls fn on primary and, if it returns an error, logs a warning and then calls fn on secondary.
func (d *FallbackDriver) try(op string, fnPrimary func() error, fnSecondary func() error) error {
	if err := fnPrimary(); err == nil {
		return nil
	} else {
		d.logger.Warn("Primary SDN driver failed, falling back", zap.String("op", op), zap.Error(err))
		// Try secondary
		if err2 := fnSecondary(); err2 != nil {
			d.logger.Error("Secondary SDN driver failed", zap.String("op", op), zap.Error(err2))
			return err2
		}
		return nil
	}
}

// getOVN tries to return an OVN driver instance from either primary or secondary
func (d *FallbackDriver) getOVN() *OVNDriver {
	if od, ok := d.primary.(*OVNDriver); ok {
		return od
	}
	if od, ok := d.secondary.(*OVNDriver); ok {
		return od
	}
	return nil
}

func (d *FallbackDriver) EnsureNetwork(n *Network, s *Subnet) error {
	// First try primary
	if err := d.primary.EnsureNetwork(n, s); err != nil {
		d.logger.Warn("Primary EnsureNetwork failed, trying secondary", zap.Error(err))
		return d.secondary.EnsureNetwork(n, s)
	}
	// Verify presence via OVN if available; if NB is unreachable, do not enforce OVN
	if ovn := d.getOVN(); ovn != nil {
		lsName := fmt.Sprintf("ls-%s", n.ID)
		// Use 'find' so not-found returns empty string instead of error
		out, err := ovn.nbctlOutput("--bare", "--columns=name", "find", "Logical_Switch", fmt.Sprintf("name=%s", lsName))
		if err != nil {
			// NB unreachable: assume plugin succeeded; skip OVN enforcement
			d.logger.Info("Skip OVN verify for EnsureNetwork (NB unreachable)", zap.Error(err))
			return nil
		}
		if strings.TrimSpace(out) == "" {
			d.logger.Warn("Primary EnsureNetwork had no effect, enforcing via OVN secondary", zap.String("ls", lsName))
			return d.secondary.EnsureNetwork(n, s)
		}
	}
	return nil
}
func (d *FallbackDriver) DeleteNetwork(n *Network) error {
	return d.try("DeleteNetwork", func() error { return d.primary.DeleteNetwork(n) }, func() error { return d.secondary.DeleteNetwork(n) })
}
func (d *FallbackDriver) EnsurePort(n *Network, s *Subnet, p *NetworkPort) error {
	return d.try("EnsurePort", func() error { return d.primary.EnsurePort(n, s, p) }, func() error { return d.secondary.EnsurePort(n, s, p) })
}
func (d *FallbackDriver) DeletePort(n *Network, p *NetworkPort) error {
	return d.try("DeletePort", func() error { return d.primary.DeletePort(n, p) }, func() error { return d.secondary.DeletePort(n, p) })
}
func (d *FallbackDriver) EnsureRouter(name string) error {
	if err := d.primary.EnsureRouter(name); err != nil {
		d.logger.Warn("Primary EnsureRouter failed, trying secondary", zap.Error(err))
		return d.secondary.EnsureRouter(name)
	}
	// Verify router exists; if NB unreachable, do not enforce OVN
	if ovn := d.getOVN(); ovn != nil {
		out, err := ovn.nbctlOutput("--bare", "--columns=name", "find", "Logical_Router", fmt.Sprintf("name=%s", name))
		if err != nil {
			d.logger.Info("Skip OVN verify for EnsureRouter (NB unreachable)", zap.Error(err))
			return nil
		}
		if strings.TrimSpace(out) == "" {
			d.logger.Warn("Primary EnsureRouter had no effect, enforcing via OVN secondary", zap.String("lr", name))
			return d.secondary.EnsureRouter(name)
		}
	}
	return nil
}
func (d *FallbackDriver) DeleteRouter(name string) error {
	return d.try("DeleteRouter", func() error { return d.primary.DeleteRouter(name) }, func() error { return d.secondary.DeleteRouter(name) })
}
func (d *FallbackDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	// Try primary first
	if err := d.primary.ConnectSubnetToRouter(router, n, s); err != nil {
		d.logger.Warn("Primary ConnectSubnetToRouter failed, trying secondary", zap.Error(err))
		return d.secondary.ConnectSubnetToRouter(router, n, s)
	}
	// Verify LSP has addresses=router and LRP has networks; if not, apply via OVN secondary
	if ovn := d.getOVN(); ovn != nil {
		lsp := fmt.Sprintf("lsp-%s-%s", router, n.ID)
		lrp := fmt.Sprintf("lrp-%s-%s", router, n.ID)
		// First verify rows exist using find
		lspOut, errLsp := ovn.nbctlOutput("--bare", "--columns=name", "find", "Logical_Switch_Port", fmt.Sprintf("name=%s", lsp))
		lrpOut, errLrp := ovn.nbctlOutput("--bare", "--columns=name", "find", "Logical_Router_Port", fmt.Sprintf("name=%s", lrp))
		if errLsp != nil || errLrp != nil {
			// NB unreachable: assume plugin succeeded; skip OVN enforcement
			d.logger.Info("Skip OVN verify for ConnectSubnetToRouter (NB unreachable)", zap.String("lsp", lsp), zap.String("lrp", lrp))
			return nil
		}
		if strings.TrimSpace(lspOut) == "" || strings.TrimSpace(lrpOut) == "" {
			d.logger.Warn("Primary ConnectSubnetToRouter did not create ports, enforcing via OVN secondary", zap.String("lsp", lsp), zap.String("lrp", lrp))
			return d.secondary.ConnectSubnetToRouter(router, n, s)
		}
		// Rows exist; verify important fields
		addr, _ := ovn.nbctlOutput("get", "Logical_Switch_Port", lsp, "addresses")
		nets, _ := ovn.nbctlOutput("get", "Logical_Router_Port", lrp, "networks")
		if strings.TrimSpace(addr) == "" || strings.TrimSpace(addr) == "[]" || strings.TrimSpace(nets) == "" || strings.TrimSpace(nets) == "[]" {
			d.logger.Warn("Primary ConnectSubnetToRouter missing addresses/networks, enforcing via OVN secondary", zap.String("lsp", lsp), zap.String("lrp", lrp))
			return d.secondary.ConnectSubnetToRouter(router, n, s)
		}
	}
	return nil
}

// coalesceErr returns the first non-nil error
// (helper removed)
func (d *FallbackDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	return d.try("DisconnectSubnetFromRouter", func() error { return d.primary.DisconnectSubnetFromRouter(router, n) }, func() error { return d.secondary.DisconnectSubnetFromRouter(router, n) })
}
func (d *FallbackDriver) SetRouterGateway(router string, externalNetwork *Network, externalSubnet *Subnet) (string, error) {
	ip := ""
	var errPrim error
	// Wrap to reuse try
	err := d.try("SetRouterGateway",
		func() error {
			ip, errPrim = d.primary.SetRouterGateway(router, externalNetwork, externalSubnet)
			return errPrim
		},
		func() error {
			var errSec error
			ip, errSec = d.secondary.SetRouterGateway(router, externalNetwork, externalSubnet)
			return errSec
		})
	return ip, err
}
func (d *FallbackDriver) ClearRouterGateway(router string, externalNetwork *Network) error {
	return d.try("ClearRouterGateway", func() error { return d.primary.ClearRouterGateway(router, externalNetwork) }, func() error { return d.secondary.ClearRouterGateway(router, externalNetwork) })
}
func (d *FallbackDriver) SetRouterSNAT(router string, enable bool, internalCIDR string, externalIP string) error {
	return d.try("SetRouterSNAT", func() error { return d.primary.SetRouterSNAT(router, enable, internalCIDR, externalIP) }, func() error { return d.secondary.SetRouterSNAT(router, enable, internalCIDR, externalIP) })
}
func (d *FallbackDriver) EnsureFIPNAT(router string, floatingIP, fixedIP string) error {
	return d.try("EnsureFIPNAT", func() error { return d.primary.EnsureFIPNAT(router, floatingIP, fixedIP) }, func() error { return d.secondary.EnsureFIPNAT(router, floatingIP, fixedIP) })
}
func (d *FallbackDriver) RemoveFIPNAT(router string, floatingIP, fixedIP string) error {
	return d.try("RemoveFIPNAT", func() error { return d.primary.RemoveFIPNAT(router, floatingIP, fixedIP) }, func() error { return d.secondary.RemoveFIPNAT(router, floatingIP, fixedIP) })
}
func (d *FallbackDriver) ReplacePortACLs(networkID, portID string, rules []ACLRule) error {
	return d.try("ReplacePortACLs", func() error { return d.primary.ReplacePortACLs(networkID, portID, rules) }, func() error { return d.secondary.ReplacePortACLs(networkID, portID, rules) })
}
func (d *FallbackDriver) EnsurePortSecurity(portID string, groups []CompiledSecurityGroup) error {
	return d.try("EnsurePortSecurity", func() error { return d.primary.EnsurePortSecurity(portID, groups) }, func() error { return d.secondary.EnsurePortSecurity(portID, groups) })
}
