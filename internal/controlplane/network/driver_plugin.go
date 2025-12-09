package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// PluginConfig holds network plugin connection parameters
type PluginConfig struct {
	Endpoint string // e.g. http://localhost:8086
}

// PluginDriver implements Driver by calling network-plugin HTTP API
type PluginDriver struct {
	logger *zap.Logger
	cfg    PluginConfig
	client *http.Client
}

func NewPluginDriver(l *zap.Logger, cfg PluginConfig) *PluginDriver {
	return &PluginDriver{
		logger: l,
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// EnsureNetwork creates a logical switch via plugin
func (d *PluginDriver) EnsureNetwork(n *Network, s *Subnet) error {
	// Create logical switch
	payload := map[string]string{
		"id":   n.ID,
		"cidr": n.CIDR,
	}
	if err := d.post("/api/v1/networks", payload); err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	// Configure DHCP if subnet has CIDR and DHCP is enabled
	if s != nil && strings.TrimSpace(s.CIDR) != "" && s.EnableDHCP {
		// Calculate gateway if not provided
		gateway := s.Gateway
		if gateway == "" {
			_, ipnet, err := net.ParseCIDR(s.CIDR)
			if err != nil {
				return fmt.Errorf("invalid CIDR %s: %w", s.CIDR, err)
			}
			ip := ipnet.IP.To4()
			if ip != nil {
				ip[3] = ip[3] + 1
				gateway = ip.String()
				s.Gateway = gateway
			}
		}

		// Calculate allocation pool if not provided
		if s.AllocationStart == "" || s.AllocationEnd == "" {
			_, ipnet, err := net.ParseCIDR(s.CIDR)
			if err == nil {
				ip := ipnet.IP.To4()
				if ip != nil {
					startIP := make(net.IP, 4)
					copy(startIP, ip)
					startIP[3] = startIP[3] + 2
					s.AllocationStart = startIP.String()

					ones, bits := ipnet.Mask.Size()
					if bits == 32 {
						hostBits := bits - ones
						numHosts := (1 << hostBits) - 2
						endIP := make(net.IP, 4)
						copy(endIP, ip)
						endIP[3] = endIP[3] + byte(numHosts)
						s.AllocationEnd = endIP.String()
					}
				}
			}
		}

		// Set DNS servers
		dnsServers := s.DNSNameservers
		if dnsServers == "" {
			dnsServers = "8.8.8.8,8.8.4.4"
			s.DNSNameservers = dnsServers
		}

		leaseTime := s.DHCPLeaseTime
		if leaseTime == 0 {
			leaseTime = 86400
		}

		dhcpPayload := map[string]interface{}{
			"network_id":       n.ID,
			"cidr":             s.CIDR,
			"gateway":          gateway,
			"dns_servers":      dnsServers,
			"allocation_start": s.AllocationStart,
			"allocation_end":   s.AllocationEnd,
			"lease_time":       leaseTime,
		}

		if err := d.post("/api/v1/dhcp", dhcpPayload); err != nil {
			return fmt.Errorf("failed to configure DHCP: %w", err)
		}

		d.logger.Info("DHCP configured successfully",
			zap.String("network", n.ID),
			zap.String("subnet", s.ID),
			zap.String("gateway", gateway))
	}

	return nil
}

// DeleteNetwork removes the logical switch
func (d *PluginDriver) DeleteNetwork(n *Network) error {
	url := fmt.Sprintf("%s/api/v1/networks/%s", d.cfg.Endpoint, n.ID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete network: status %d", resp.StatusCode)
	}

	return nil
}

// EnsurePort creates a logical switch port
func (d *PluginDriver) EnsurePort(n *Network, s *Subnet, p *NetworkPort) error {
	first := ""
	subnetID := ""
	if len(p.FixedIPs) > 0 {
		first = p.FixedIPs[0].IP
		subnetID = p.FixedIPs[0].SubnetID
	}
	// Fallback to port.SubnetID if FixedIPs doesn't have it
	if subnetID == "" && p.SubnetID != "" {
		subnetID = p.SubnetID
	}

	payload := map[string]string{
		"network_id":  n.ID,
		"port_id":     p.ID,
		"mac_address": p.MACAddress,
		"ip_address":  first,
		"subnet_id":   subnetID,
	}

	if err := d.post("/api/v1/ports", payload); err != nil {
		return fmt.Errorf("failed to create port: %w", err)
	}

	return nil
}

// DeletePort removes a logical switch port
func (d *PluginDriver) DeletePort(n *Network, p *NetworkPort) error {
	url := fmt.Sprintf("%s/api/v1/ports/%s", d.cfg.Endpoint, p.ID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete port: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete port: status %d", resp.StatusCode)
	}

	return nil
}

// EnsureRouter creates a logical router (stub for now)
func (d *PluginDriver) EnsureRouter(name string) error {
	payload := map[string]string{"name": name}
	return d.post("/api/v1/routers", payload)
}

// DeleteRouter deletes a logical router (stub for now)
func (d *PluginDriver) DeleteRouter(name string) error {
	url := fmt.Sprintf("%s/api/v1/routers/%s", d.cfg.Endpoint, name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("plugin returned status %d", resp.StatusCode)
	}
	return nil
}

// ConnectSubnetToRouter connects subnet to router (stub for now)
func (d *PluginDriver) ConnectSubnetToRouter(router string, n *Network, s *Subnet) error {
	payload := map[string]string{
		"network_id": n.ID,
		"cidr":       s.CIDR,
		"gateway":    s.Gateway,
	}
	path := fmt.Sprintf("/api/v1/routers/%s/connect-subnet", router)
	return d.post(path, payload)
}

// DisconnectSubnetFromRouter disconnects subnet from router (stub for now)
func (d *PluginDriver) DisconnectSubnetFromRouter(router string, n *Network) error {
	payload := map[string]string{"network_id": n.ID}
	path := fmt.Sprintf("/api/v1/routers/%s/disconnect-subnet", router)
	return d.post(path, payload)
}

// SetRouterGateway sets external gateway for router (stub for now)
func (d *PluginDriver) SetRouterGateway(router string, externalNetwork *Network, externalSubnet *Subnet) (string, error) {
	payload := map[string]string{
		"external_network_id": externalNetwork.ID,
		"external_cidr":       externalSubnet.CIDR,
		"external_gateway":    externalSubnet.Gateway,
	}
	path := fmt.Sprintf("/api/v1/routers/%s/set-gateway", router)
	if err := d.post(path, payload); err != nil {
		return "", err
	}
	// Derive allocated IP like plugin does: second usable address
	ip, _, err := net.ParseCIDR(externalSubnet.CIDR)
	if err != nil {
		return "", fmt.Errorf("invalid external subnet CIDR: %w", err)
	}
	v4 := ip.To4()
	if v4 == nil {
		return "", fmt.Errorf("only IPv4 is supported")
	}
	// second usable (gateway is first)
	rIP := inc(v4)
	rIP = inc(rIP)
	return rIP.String(), nil
}

// ClearRouterGateway clears external gateway for router (stub for now)
func (d *PluginDriver) ClearRouterGateway(router string, externalNetwork *Network) error {
	payload := map[string]string{"external_network_id": externalNetwork.ID}
	path := fmt.Sprintf("/api/v1/routers/%s/clear-gateway", router)
	return d.post(path, payload)
}

// SetRouterSNAT enables/disables SNAT for router (stub for now)
func (d *PluginDriver) SetRouterSNAT(router string, enable bool, internalCIDR string, externalIP string) error {
	payload := map[string]interface{}{
		"enable":        enable,
		"internal_cidr": internalCIDR,
		"external_ip":   externalIP,
	}
	path := fmt.Sprintf("/api/v1/routers/%s/snat", router)
	return d.post(path, payload)
}

// EnsureFIPNAT configures floating IP NAT (stub for now)
func (d *PluginDriver) EnsureFIPNAT(router string, floatingIP, fixedIP string) error {
	d.logger.Debug("EnsureFIPNAT called")
	// TODO: Implement NAT via plugin
	return nil
}

// RemoveFIPNAT removes floating IP NAT (stub for now)
func (d *PluginDriver) RemoveFIPNAT(router string, floatingIP, fixedIP string) error {
	d.logger.Debug("RemoveFIPNAT called")
	// TODO: Implement NAT removal via plugin
	return nil
}

// ReplacePortACLs updates port ACLs (stub for now)
func (d *PluginDriver) ReplacePortACLs(networkID, portID string, rules []ACLRule) error {
	d.logger.Debug("ReplacePortACLs called")
	// TODO: Implement ACL management via plugin
	return nil
}

// EnsurePortSecurity applies security groups to port (stub for now)
func (d *PluginDriver) EnsurePortSecurity(portID string, groups []CompiledSecurityGroup) error {
	d.logger.Debug("EnsurePortSecurity called")
	// TODO: Implement security group management via plugin
	return nil
}

// small helper for IPv4 increment
func inc(ip net.IP) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)
	for i := len(out) - 1; i >= 0; i-- {
		out[i]++
		if out[i] != 0 {
			break
		}
	}
	return out
}

// post sends a POST request to the plugin
func (d *PluginDriver) post(path string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := d.cfg.Endpoint + path
	d.logger.Debug("POST to plugin", zap.String("url", url), zap.String("payload", string(data)))

	resp, err := d.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("plugin returned status %d", resp.StatusCode)
	}

	return nil
}
