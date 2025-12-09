package lite

import "time"

// Driver abstracts hypervisor operations (libvirt expected implementation).
type Driver interface {
	CreateVM(CreateVMRequest) (*VM, error)
	DeleteVM(id string, force bool) error
	StartVM(id string) error
	StopVM(id string, force bool) error
	RebootVM(id string, force bool) error
	ConsoleURL(id string, ttl time.Duration) (string, error)
	VMStatus(id string) (exists bool, running bool)
	Status() NodeStatus
}
