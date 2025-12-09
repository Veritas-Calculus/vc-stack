package lite

import "time"

// VM represents a virtual machine metadata from the node perspective.
type VM struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	VCPUs     int       `json:"vcpus"`
	MemoryMB  int       `json:"memory_mb"`
	DiskGB    int       `json:"disk_gb"`
	Image     string    `json:"image"`
	Status    string    `json:"status"` // creating, running, stopped, deleting, error
	Power     string    `json:"power"`  // running, shutdown
	Hostname  string    `json:"hostname"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateVMRequest is the minimal payload to create a VM.
type CreateVMRequest struct {
	Name     string `json:"name" binding:"required"`
	VCPUs    int    `json:"vcpus" binding:"required"`
	MemoryMB int    `json:"memory_mb" binding:"required"`
	DiskGB   int    `json:"disk_gb" binding:"required"`
	Image    string `json:"image"`
	UEFI     bool   `json:"uefi"`
	TPM      bool   `json:"tpm"`
	// Network configuration
	NetworkID string `json:"network_id"` // Network UUID from net_networks table
	// Network interfaces (optional). If empty, one interface on default network is attached.
	Nics []Nic `json:"nics"`
	// Optional: Ceph RBD images instead of local files
	// RootRBDImage format: <pool>/<image>@<snap> (snap optional)
	RootRBDImage string `json:"root_rbd_image"`
	// IsoRBDImage format: <pool>/<image>@<snap> (snap optional)
	IsoRBDImage string `json:"iso_rbd_image"`
	// If using local ISO file (fallback)
	ISO string `json:"iso"`
	// Cloud-init support
	SSHAuthorizedKey string `json:"ssh_authorized_key"`
	UserData         string `json:"user_data"`
}

// Nic describes a VM NIC configuration
type Nic struct {
	MAC    string `json:"mac"`
	PortID string `json:"port_id"` // OVN/OVS port ID for virtualport interfaceid
}

// NodeStatus basic capacity and usage snapshot.
type NodeStatus struct {
	CPUsTotal   int     `json:"cpus_total"`
	CPUsUsed    int     `json:"cpus_used"`
	RAMMBTotal  int     `json:"ram_mb_total"`
	RAMMBUsed   int     `json:"ram_mb_used"`
	DiskGBTotal int     `json:"disk_gb_total"`
	DiskGBUsed  int     `json:"disk_gb_used"`
	UptimeSec   int64   `json:"uptime_sec"`
	LoadAvg1    float64 `json:"load1"`
}
