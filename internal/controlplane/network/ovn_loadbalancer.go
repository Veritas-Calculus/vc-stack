// Package network provides OVN load balancer support.
package network

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// OVNLoadBalancer represents an OVN load balancer.
type OVNLoadBalancer struct {
	Name      string
	UUID      string
	VIP       string
	Protocol  string // tcp or udp
	Backends  []string
	Algorithm string // round-robin, source, etc.
}

// OVNLoadBalancerManager manages OVN load balancers.
type OVNLoadBalancerManager struct {
	driver *OVNDriver
	logger *zap.Logger
	mu     sync.RWMutex

	loadBalancers map[string]*OVNLoadBalancer
}

// NewOVNLoadBalancerManager creates a new load balancer manager.
func NewOVNLoadBalancerManager(driver *OVNDriver, logger *zap.Logger) *OVNLoadBalancerManager {
	return &OVNLoadBalancerManager{
		driver:        driver,
		logger:        logger,
		loadBalancers: make(map[string]*OVNLoadBalancer),
	}
}

// CreateLoadBalancer creates an OVN load balancer.
func (m *OVNLoadBalancerManager) CreateLoadBalancer(ctx context.Context, name, vip, protocol string, backends []string) (*OVNLoadBalancer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.loadBalancers[name]; exists {
		return nil, fmt.Errorf("load balancer %s already exists", name)
	}

	// Build VIP string: "VIP:port"
	vips := m.buildVIPString(vip, backends)

	// Create load balancer.
	args := []string{
		"--may-exist", "lb-add", name, vips, protocol,
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("create load balancer: %v, output: %s", err, string(output))
	}

	// Get UUID.
	uuid, err := m.getLoadBalancerUUID(name)
	if err != nil {
		m.logger.Warn("Failed to get load balancer UUID", zap.Error(err))
		uuid = ""
	}

	lb := &OVNLoadBalancer{
		Name:      name,
		UUID:      uuid,
		VIP:       vip,
		Protocol:  protocol,
		Backends:  backends,
		Algorithm: "round-robin",
	}

	m.loadBalancers[name] = lb

	m.logger.Info("Created load balancer",
		zap.String("name", name),
		zap.String("vip", vip),
		zap.String("protocol", protocol),
		zap.Strings("backends", backends))

	return lb, nil
}

// DeleteLoadBalancer deletes an OVN load balancer.
func (m *OVNLoadBalancerManager) DeleteLoadBalancer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.loadBalancers[name]; !exists {
		return fmt.Errorf("load balancer %s not found", name)
	}

	args := []string{
		"--if-exists", "lb-del", name,
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete load balancer: %v, output: %s", err, string(output))
	}

	delete(m.loadBalancers, name)

	m.logger.Info("Deleted load balancer", zap.String("name", name))
	return nil
}

// UpdateBackends updates load balancer backends.
func (m *OVNLoadBalancerManager) UpdateBackends(ctx context.Context, name string, backends []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lb, exists := m.loadBalancers[name]
	if !exists {
		return fmt.Errorf("load balancer %s not found", name)
	}

	// Build new VIP string.
	vips := m.buildVIPString(lb.VIP, backends)

	// Update load balancer.
	args := []string{
		"lb-add", name, vips, lb.Protocol,
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update load balancer: %v, output: %s", err, string(output))
	}

	lb.Backends = backends

	m.logger.Info("Updated load balancer backends",
		zap.String("name", name),
		zap.Strings("backends", backends))

	return nil
}

// AttachToRouter attaches load balancer to logical router.
func (m *OVNLoadBalancerManager) AttachToRouter(ctx context.Context, lbName, routerName string) error {
	args := []string{
		"lr-lb-add", routerName, lbName,
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("attach to router: %v, output: %s", err, string(output))
	}

	m.logger.Info("Attached load balancer to router",
		zap.String("lb", lbName),
		zap.String("router", routerName))

	return nil
}

// DetachFromRouter detaches load balancer from logical router.
func (m *OVNLoadBalancerManager) DetachFromRouter(ctx context.Context, lbName, routerName string) error {
	args := []string{
		"--if-exists", "lr-lb-del", routerName, lbName,
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("detach from router: %v, output: %s", err, string(output))
	}

	m.logger.Info("Detached load balancer from router",
		zap.String("lb", lbName),
		zap.String("router", routerName))

	return nil
}

// AttachToSwitch attaches load balancer to logical switch.
func (m *OVNLoadBalancerManager) AttachToSwitch(ctx context.Context, lbName, switchName string) error {
	args := []string{
		"ls-lb-add", switchName, lbName,
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("attach to switch: %v, output: %s", err, string(output))
	}

	m.logger.Info("Attached load balancer to switch",
		zap.String("lb", lbName),
		zap.String("switch", switchName))

	return nil
}

// DetachFromSwitch detaches load balancer from logical switch.
func (m *OVNLoadBalancerManager) DetachFromSwitch(ctx context.Context, lbName, switchName string) error {
	args := []string{
		"--if-exists", "ls-lb-del", switchName, lbName,
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("detach from switch: %v, output: %s", err, string(output))
	}

	m.logger.Info("Detached load balancer from switch",
		zap.String("lb", lbName),
		zap.String("switch", switchName))

	return nil
}

// GetLoadBalancer retrieves load balancer by name.
func (m *OVNLoadBalancerManager) GetLoadBalancer(name string) (*OVNLoadBalancer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lb, exists := m.loadBalancers[name]
	if !exists {
		return nil, fmt.Errorf("load balancer %s not found", name)
	}

	return lb, nil
}

// ListLoadBalancers lists all load balancers.
func (m *OVNLoadBalancerManager) ListLoadBalancers() []*OVNLoadBalancer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lbs := make([]*OVNLoadBalancer, 0, len(m.loadBalancers))
	for _, lb := range m.loadBalancers {
		lbs = append(lbs, lb)
	}

	return lbs
}

// buildVIPString builds VIP string for OVN load balancer.
// Format: "VIP:port=backend1:port,backend2:port,..."
func (m *OVNLoadBalancerManager) buildVIPString(vip string, backends []string) string {
	return fmt.Sprintf("%s=%s", vip, strings.Join(backends, ","))
}

// getLoadBalancerUUID retrieves load balancer UUID.
func (m *OVNLoadBalancerManager) getLoadBalancerUUID(name string) (string, error) {
	args := []string{
		"--format=csv", "--no-headings", "--columns=_uuid",
		"find", "Load_Balancer", fmt.Sprintf("name=%s", name),
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("get load balancer UUID: %v", err)
	}

	uuid := strings.TrimSpace(string(output))
	if uuid == "" {
		return "", fmt.Errorf("load balancer UUID not found")
	}

	return uuid, nil
}

// SetAlgorithm sets load balancing algorithm.
func (m *OVNLoadBalancerManager) SetAlgorithm(ctx context.Context, name, algorithm string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lb, exists := m.loadBalancers[name]
	if !exists {
		return fmt.Errorf("load balancer %s not found", name)
	}

	// Set algorithm via external_ids or options.
	args := []string{
		"set", "Load_Balancer", name,
		fmt.Sprintf("options:hash_fields=%s", algorithm),
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.logger.Warn("Failed to set algorithm",
			zap.Error(err),
			zap.String("output", string(output)))
	}

	lb.Algorithm = algorithm

	m.logger.Info("Set load balancer algorithm",
		zap.String("name", name),
		zap.String("algorithm", algorithm))

	return nil
}

// EnableHealthCheck enables health checking for backends.
func (m *OVNLoadBalancerManager) EnableHealthCheck(ctx context.Context, name string, interval, timeout int) error {
	// Create health check configuration.
	args := []string{
		"set", "Load_Balancer", name,
		fmt.Sprintf("options:health_check_interval=%d", interval),
		fmt.Sprintf("options:health_check_timeout=%d", timeout),
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable health check: %v, output: %s", err, string(output))
	}

	m.logger.Info("Enabled health check",
		zap.String("name", name),
		zap.Int("interval", interval),
		zap.Int("timeout", timeout))

	return nil
}

// SyncLoadBalancers synchronizes load balancers from OVN.
func (m *OVNLoadBalancerManager) SyncLoadBalancers(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// List all load balancers from OVN.
	args := []string{
		"--format=csv", "--no-headings", "--columns=name,vips,protocol",
		"list", "Load_Balancer",
	}

	cmd := exec.Command("ovn-nbctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("list load balancers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	ovnLBs := make(map[string]bool)

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 1 {
			name := strings.TrimSpace(parts[0])
			ovnLBs[name] = true
		}
	}

	// Remove load balancers not in OVN.
	for name := range m.loadBalancers {
		if !ovnLBs[name] {
			m.logger.Info("Removing stale load balancer", zap.String("name", name))
			delete(m.loadBalancers, name)
		}
	}

	m.logger.Info("Load balancer sync completed",
		zap.Int("managed", len(m.loadBalancers)),
		zap.Int("ovn", len(ovnLBs)))

	return nil
}
