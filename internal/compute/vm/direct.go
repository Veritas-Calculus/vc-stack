// Package lite provides direct method access for in-process callers.
// These exported methods allow the compute service (in the same process)
// to call lite operations directly, avoiding the overhead and fragility
// of HTTP self-calls through localhost.
package vm

import (
	"time"

	"go.uber.org/zap"
)

// CreateVMDirect creates a VM and returns its metadata without going through HTTP.
// This is the preferred method for in-process callers (e.g., node/compute).
func (s *Service) CreateVMDirect(req CreateVMRequest) (*VM, error) {
	s.logger.Info("CreateVMDirect called", zap.String("name", req.Name))

	vm, err := s.drv.CreateVM(req)
	if err != nil {
		return nil, err
	}

	s.met.Inc(MVMCreateTotal, 1)
	s.mu.Lock()
	s.vms[vm.ID] = vm
	s.mu.Unlock()

	return vm, nil
}

// DeleteVMDirect deletes a VM by ID without going through HTTP.
func (s *Service) DeleteVMDirect(id string, force bool) error {
	if err := s.drv.DeleteVM(id, force); err != nil {
		return err
	}

	s.met.Inc(MVMDeleteTotal, 1)
	s.mu.Lock()
	delete(s.vms, id)
	s.mu.Unlock()

	return nil
}

// StartVMDirect starts a VM by ID without going through HTTP.
func (s *Service) StartVMDirect(id string) error {
	return s.drv.StartVM(id)
}

// StopVMDirect stops a VM by ID without going through HTTP.
func (s *Service) StopVMDirect(id string, force bool) error {
	return s.drv.StopVM(id, force)
}

// RebootVMDirect reboots a VM by ID without going through HTTP.
func (s *Service) RebootVMDirect(id string, force bool) error {
	return s.drv.RebootVM(id, force)
}

// VMStatusDirect returns the existence and running state of a VM.
func (s *Service) VMStatusDirect(id string) (exists bool, running bool) {
	return s.drv.VMStatus(id)
}

// ConsoleURLDirect returns the console URL for a VM.
func (s *Service) ConsoleURLDirect(id string, ttl time.Duration) (string, error) {
	return s.drv.ConsoleURL(id, ttl)
}

// GetConsoleTicket creates a console ticket and returns the token.
// This allows the compute service to generate console tickets directly.
func (s *Service) GetConsoleTicket(id string, ttl time.Duration) (token string, err error) {
	vncURL, err := s.drv.ConsoleURL(id, ttl)
	if err != nil {
		return "", err
	}

	token = genToken(16)
	s.mu.Lock()
	s.tokens[token] = consoleToken{VNCAddr: vncURL, ExpiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()

	return token, nil
}
