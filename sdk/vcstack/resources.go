package vcstack

import (
	"context"
	"fmt"
	"net/http"
)

// ──────────────────────────────────────────────────────────────────────
// Instance (Compute)
// ──────────────────────────────────────────────────────────────────────

// Instance represents a virtual machine instance.
type Instance struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	UUID       string `json:"uuid"`
	VMID       string `json:"vm_id,omitempty"`
	Status     string `json:"status"`
	PowerState string `json:"power_state,omitempty"`
	FlavorID   int    `json:"flavor_id,omitempty"`
	ImageID    int    `json:"image_id,omitempty"`
	HostID     string `json:"host_id,omitempty"`
	ProjectID  int    `json:"project_id,omitempty"`
	UserID     int    `json:"user_id,omitempty"`
	IPAddress  string `json:"ip_address,omitempty"`
	FloatingIP string `json:"floating_ip,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

// CreateInstanceRequest specifies parameters for creating a new instance.
type CreateInstanceRequest struct {
	Name      string `json:"name"`
	FlavorID  int    `json:"flavor_id"`
	ImageID   int    `json:"image_id"`
	NetworkID int    `json:"network_id,omitempty"`
	SSHKeyID  int    `json:"ssh_key_id,omitempty"`
	UserData  string `json:"user_data,omitempty"`
	Count     int    `json:"count,omitempty"`
}

// InstanceClient handles instance operations.
type InstanceClient struct{ c *Client }

// List returns all instances.
func (ic *InstanceClient) List(ctx context.Context) ([]Instance, error) {
	var resp struct {
		Instances []Instance `json:"instances"`
	}
	if err := ic.c.do(ctx, http.MethodGet, "/v1/instances", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Instances, nil
}

// Get returns a single instance by ID.
func (ic *InstanceClient) Get(ctx context.Context, id string) (*Instance, error) {
	var resp struct {
		Instance Instance `json:"instance"`
	}
	if err := ic.c.do(ctx, http.MethodGet, "/v1/instances/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Instance, nil
}

// Create creates a new instance.
func (ic *InstanceClient) Create(ctx context.Context, req *CreateInstanceRequest) (*Instance, error) {
	var resp struct {
		Instance Instance `json:"instance"`
	}
	if err := ic.c.do(ctx, http.MethodPost, "/v1/instances", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Instance, nil
}

// Delete deletes an instance.
func (ic *InstanceClient) Delete(ctx context.Context, id string) error {
	return ic.c.do(ctx, http.MethodDelete, "/v1/instances/"+id, nil, nil)
}

// Action performs a power action on an instance (start, stop, reboot, etc.).
func (ic *InstanceClient) Action(ctx context.Context, id, action string) error {
	return ic.c.do(ctx, http.MethodPost, "/v1/instances/"+id+"/"+action, nil, nil)
}

// ──────────────────────────────────────────────────────────────────────
// Flavor
// ──────────────────────────────────────────────────────────────────────

// Flavor represents a VM resource template.
type Flavor struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	VCPUs int    `json:"vcpus"`
	RAM   int    `json:"ram"` // MB
	Disk  int    `json:"disk,omitempty"`
}

// FlavorClient handles flavor operations.
type FlavorClient struct{ c *Client }

// List returns all flavors.
func (fc *FlavorClient) List(ctx context.Context) ([]Flavor, error) {
	var resp struct {
		Flavors []Flavor `json:"flavors"`
	}
	if err := fc.c.do(ctx, http.MethodGet, "/v1/flavors", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Flavors, nil
}

// Get returns a single flavor by ID.
func (fc *FlavorClient) Get(ctx context.Context, id string) (*Flavor, error) {
	var resp struct {
		Flavor Flavor `json:"flavor"`
	}
	if err := fc.c.do(ctx, http.MethodGet, "/v1/flavors/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Flavor, nil
}

// ──────────────────────────────────────────────────────────────────────
// Image
// ──────────────────────────────────────────────────────────────────────

// Image represents an OS image.
type Image struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Size       int64  `json:"size,omitempty"`
	MinDisk    int    `json:"min_disk,omitempty"`
	DiskFormat string `json:"disk_format,omitempty"`
	OwnerID    int    `json:"owner_id,omitempty"`
}

// ImageClient handles image operations.
type ImageClient struct{ c *Client }

// List returns all images.
func (imc *ImageClient) List(ctx context.Context) ([]Image, error) {
	var resp struct {
		Images []Image `json:"images"`
	}
	if err := imc.c.do(ctx, http.MethodGet, "/v1/images", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Images, nil
}

// Get returns a single image by ID.
func (imc *ImageClient) Get(ctx context.Context, id string) (*Image, error) {
	var resp struct {
		Image Image `json:"image"`
	}
	if err := imc.c.do(ctx, http.MethodGet, "/v1/images/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Image, nil
}

// ──────────────────────────────────────────────────────────────────────
// Volume (Block Storage)
// ──────────────────────────────────────────────────────────────────────

// Volume represents a block storage volume.
type Volume struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	SizeGB     int    `json:"size_gb"`
	Status     string `json:"status"`
	VolumeType string `json:"volume_type,omitempty"`
	ProjectID  int    `json:"project_id,omitempty"`
	InstanceID int    `json:"instance_id,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// CreateVolumeRequest specifies parameters for creating a volume.
type CreateVolumeRequest struct {
	Name       string `json:"name"`
	SizeGB     int    `json:"size_gb"`
	VolumeType string `json:"volume_type,omitempty"`
}

// VolumeClient handles volume operations.
type VolumeClient struct{ c *Client }

// List returns all volumes.
func (vc *VolumeClient) List(ctx context.Context) ([]Volume, error) {
	var resp struct {
		Volumes []Volume `json:"volumes"`
	}
	if err := vc.c.do(ctx, http.MethodGet, "/v1/storage/volumes", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Volumes, nil
}

// Get returns a single volume by ID.
func (vc *VolumeClient) Get(ctx context.Context, id string) (*Volume, error) {
	var resp struct {
		Volume Volume `json:"volume"`
	}
	if err := vc.c.do(ctx, http.MethodGet, "/v1/storage/volumes/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Volume, nil
}

// Create creates a new volume.
func (vc *VolumeClient) Create(ctx context.Context, req *CreateVolumeRequest) (*Volume, error) {
	var resp struct {
		Volume Volume `json:"volume"`
	}
	if err := vc.c.do(ctx, http.MethodPost, "/v1/storage/volumes", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Volume, nil
}

// Delete deletes a volume.
func (vc *VolumeClient) Delete(ctx context.Context, id string) error {
	return vc.c.do(ctx, http.MethodDelete, "/v1/storage/volumes/"+id, nil, nil)
}

// Attach attaches a volume to an instance.
func (vc *VolumeClient) Attach(ctx context.Context, id string, instanceID int) error {
	body := map[string]int{"instance_id": instanceID}
	return vc.c.do(ctx, http.MethodPost, "/v1/storage/volumes/"+id+"/attach", body, nil)
}

// Detach detaches a volume from an instance.
func (vc *VolumeClient) Detach(ctx context.Context, id string, instanceID string) error {
	body := map[string]string{"instance_id": instanceID}
	return vc.c.do(ctx, http.MethodPost, "/v1/storage/volumes/"+id+"/detach", body, nil)
}

// ──────────────────────────────────────────────────────────────────────
// Network
// ──────────────────────────────────────────────────────────────────────

// Network represents a virtual network.
type Network struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	NetworkType    string `json:"network_type,omitempty"`
	SegmentationID int    `json:"segmentation_id,omitempty"`
	External       bool   `json:"external"`
	Shared         bool   `json:"shared"`
	TenantID       string `json:"tenant_id,omitempty"`
	Status         string `json:"status,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
}

// CreateNetworkRequest specifies parameters for creating a network.
type CreateNetworkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	NetworkType string `json:"network_type,omitempty"`
	TenantID    string `json:"tenant_id,omitempty"`
	External    bool   `json:"external,omitempty"`
	Shared      bool   `json:"shared,omitempty"`
}

// NetworkClient handles network operations.
type NetworkClient struct{ c *Client }

// List returns all networks.
func (nc *NetworkClient) List(ctx context.Context) ([]Network, error) {
	var resp struct {
		Networks []Network `json:"networks"`
	}
	if err := nc.c.do(ctx, http.MethodGet, "/v1/networks", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Networks, nil
}

// Get returns a single network by ID.
func (nc *NetworkClient) Get(ctx context.Context, id string) (*Network, error) {
	var resp struct {
		Network Network `json:"network"`
	}
	if err := nc.c.do(ctx, http.MethodGet, "/v1/networks/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Network, nil
}

// Create creates a new network.
func (nc *NetworkClient) Create(ctx context.Context, req *CreateNetworkRequest) (*Network, error) {
	var resp struct {
		Network Network `json:"network"`
	}
	if err := nc.c.do(ctx, http.MethodPost, "/v1/networks", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Network, nil
}

// Delete deletes a network.
func (nc *NetworkClient) Delete(ctx context.Context, id string) error {
	return nc.c.do(ctx, http.MethodDelete, "/v1/networks/"+id, nil, nil)
}

// ──────────────────────────────────────────────────────────────────────
// Subnet
// ──────────────────────────────────────────────────────────────────────

// Subnet represents a subnet within a network.
type Subnet struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CIDR      string `json:"cidr"`
	Gateway   string `json:"gateway,omitempty"`
	NetworkID int    `json:"network_id"`
	TenantID  string `json:"tenant_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// CreateSubnetRequest specifies parameters for creating a subnet.
type CreateSubnetRequest struct {
	Name      string `json:"name"`
	CIDR      string `json:"cidr"`
	Gateway   string `json:"gateway,omitempty"`
	NetworkID int    `json:"network_id"`
	TenantID  string `json:"tenant_id,omitempty"`
}

// SubnetClient handles subnet operations.
type SubnetClient struct{ c *Client }

// List returns all subnets.
func (sc *SubnetClient) List(ctx context.Context) ([]Subnet, error) {
	var resp struct {
		Subnets []Subnet `json:"subnets"`
	}
	if err := sc.c.do(ctx, http.MethodGet, "/v1/subnets", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Subnets, nil
}

// Create creates a new subnet.
func (sc *SubnetClient) Create(ctx context.Context, req *CreateSubnetRequest) (*Subnet, error) {
	var resp struct {
		Subnet Subnet `json:"subnet"`
	}
	if err := sc.c.do(ctx, http.MethodPost, "/v1/subnets", req, &resp); err != nil {
		return nil, err
	}
	return &resp.Subnet, nil
}

// Delete deletes a subnet.
func (sc *SubnetClient) Delete(ctx context.Context, id string) error {
	return sc.c.do(ctx, http.MethodDelete, "/v1/subnets/"+id, nil, nil)
}

// ──────────────────────────────────────────────────────────────────────
// Security Group
// ──────────────────────────────────────────────────────────────────────

// SecurityGroup represents a network security group.
type SecurityGroup struct {
	ID          int                 `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	TenantID    string              `json:"tenant_id,omitempty"`
	Rules       []SecurityGroupRule `json:"rules,omitempty"`
	CreatedAt   string              `json:"created_at,omitempty"`
}

// SecurityGroupRule represents a rule within a security group.
type SecurityGroupRule struct {
	ID              int    `json:"id"`
	Direction       string `json:"direction"` // ingress or egress
	EtherType       string `json:"ether_type,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
	PortRangeMin    int    `json:"port_range_min,omitempty"`
	PortRangeMax    int    `json:"port_range_max,omitempty"`
	RemoteIPPrefix  string `json:"remote_ip_prefix,omitempty"`
	RemoteGroupID   int    `json:"remote_group_id,omitempty"`
	SecurityGroupID int    `json:"security_group_id"`
}

// CreateSecurityGroupRequest specifies parameters for creating a security group.
type CreateSecurityGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TenantID    string `json:"tenant_id,omitempty"`
}

// SecurityGroupClient handles security group operations.
type SecurityGroupClient struct{ c *Client }

// List returns all security groups.
func (sgc *SecurityGroupClient) List(ctx context.Context) ([]SecurityGroup, error) {
	var resp struct {
		SecurityGroups []SecurityGroup `json:"security_groups"`
	}
	if err := sgc.c.do(ctx, http.MethodGet, "/v1/security-groups", nil, &resp); err != nil {
		return nil, err
	}
	return resp.SecurityGroups, nil
}

// Get returns a single security group.
func (sgc *SecurityGroupClient) Get(ctx context.Context, id string) (*SecurityGroup, error) {
	var resp struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	if err := sgc.c.do(ctx, http.MethodGet, "/v1/security-groups/"+id, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.SecurityGroup, nil
}

// Create creates a new security group.
func (sgc *SecurityGroupClient) Create(ctx context.Context, req *CreateSecurityGroupRequest) (*SecurityGroup, error) {
	var resp struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	if err := sgc.c.do(ctx, http.MethodPost, "/v1/security-groups", req, &resp); err != nil {
		return nil, err
	}
	return &resp.SecurityGroup, nil
}

// Delete deletes a security group.
func (sgc *SecurityGroupClient) Delete(ctx context.Context, id string) error {
	return sgc.c.do(ctx, http.MethodDelete, "/v1/security-groups/"+id, nil, nil)
}

// AddRule adds a rule to a security group.
func (sgc *SecurityGroupClient) AddRule(ctx context.Context, groupID string, rule *SecurityGroupRule) (*SecurityGroupRule, error) {
	var resp struct {
		Rule SecurityGroupRule `json:"rule"`
	}
	if err := sgc.c.do(ctx, http.MethodPost, "/v1/security-groups/"+groupID+"/rules", rule, &resp); err != nil {
		return nil, err
	}
	return &resp.Rule, nil
}

// DeleteRule removes a rule from a security group.
func (sgc *SecurityGroupClient) DeleteRule(ctx context.Context, groupID, ruleID string) error {
	return sgc.c.do(ctx, http.MethodDelete, "/v1/security-groups/"+groupID+"/rules/"+ruleID, nil, nil)
}

// ──────────────────────────────────────────────────────────────────────
// Floating IP
// ──────────────────────────────────────────────────────────────────────

// FloatingIP represents a public floating IP address.
type FloatingIP struct {
	ID         int    `json:"id"`
	Address    string `json:"address"`
	Status     string `json:"status"`
	InstanceID int    `json:"instance_id,omitempty"`
	NetworkID  int    `json:"network_id,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// FloatingIPClient handles floating IP operations.
type FloatingIPClient struct{ c *Client }

// List returns all floating IPs.
func (fic *FloatingIPClient) List(ctx context.Context) ([]FloatingIP, error) {
	var resp struct {
		FloatingIPs []FloatingIP `json:"floating_ips"`
	}
	if err := fic.c.do(ctx, http.MethodGet, "/v1/floating-ips", nil, &resp); err != nil {
		return nil, err
	}
	return resp.FloatingIPs, nil
}

// Create allocates a new floating IP.
func (fic *FloatingIPClient) Create(ctx context.Context, networkID int, tenantID string) (*FloatingIP, error) {
	body := map[string]interface{}{
		"network_id": networkID,
		"tenant_id":  tenantID,
	}
	var resp struct {
		FloatingIP FloatingIP `json:"floating_ip"`
	}
	if err := fic.c.do(ctx, http.MethodPost, "/v1/floating-ips", body, &resp); err != nil {
		return nil, err
	}
	return &resp.FloatingIP, nil
}

// Associate associates a floating IP with an instance.
func (fic *FloatingIPClient) Associate(ctx context.Context, fipID string, instanceID int) error {
	body := map[string]int{"instance_id": instanceID}
	return fic.c.do(ctx, http.MethodPost, "/v1/floating-ips/"+fipID+"/associate", body, nil)
}

// Disassociate removes a floating IP from an instance.
func (fic *FloatingIPClient) Disassociate(ctx context.Context, fipID string) error {
	return fic.c.do(ctx, http.MethodPost, "/v1/floating-ips/"+fipID+"/disassociate", nil, nil)
}

// Delete releases a floating IP.
func (fic *FloatingIPClient) Delete(ctx context.Context, id string) error {
	return fic.c.do(ctx, http.MethodDelete, "/v1/floating-ips/"+id, nil, nil)
}

// ──────────────────────────────────────────────────────────────────────
// SSH Key
// ──────────────────────────────────────────────────────────────────────

// SSHKey represents an SSH public key.
type SSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// SSHKeyClient handles SSH key operations.
type SSHKeyClient struct{ c *Client }

// List returns all SSH keys.
func (skc *SSHKeyClient) List(ctx context.Context) ([]SSHKey, error) {
	var resp struct {
		SSHKeys []SSHKey `json:"ssh_keys"`
	}
	if err := skc.c.do(ctx, http.MethodGet, "/v1/ssh-keys", nil, &resp); err != nil {
		return nil, err
	}
	return resp.SSHKeys, nil
}

// Create creates a new SSH key.
func (skc *SSHKeyClient) Create(ctx context.Context, name, publicKey string) (*SSHKey, error) {
	body := map[string]string{"name": name, "public_key": publicKey}
	var resp struct {
		SSHKey SSHKey `json:"ssh_key"`
	}
	if err := skc.c.do(ctx, http.MethodPost, "/v1/ssh-keys", body, &resp); err != nil {
		return nil, err
	}
	return &resp.SSHKey, nil
}

// Delete deletes an SSH key.
func (skc *SSHKeyClient) Delete(ctx context.Context, id string) error {
	return skc.c.do(ctx, http.MethodDelete, "/v1/ssh-keys/"+id, nil, nil)
}

// ──────────────────────────────────────────────────────────────────────
// Service Account
// ──────────────────────────────────────────────────────────────────────

// ServiceAccount represents an API service account.
type ServiceAccount struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AccessKeyID string `json:"access_key_id"`
	IsActive    bool   `json:"is_active"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// CreateServiceAccountRequest specifies parameters for creating a service account.
type CreateServiceAccountRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ExpiresIn   string `json:"expires_in,omitempty"`
}

// CreateServiceAccountResponse includes the one-time secret key.
type CreateServiceAccountResponse struct {
	ServiceAccount ServiceAccount `json:"service_account"`
	AccessKeyID    string         `json:"access_key_id"`
	SecretKey      string         `json:"secret_key"`
}

// ServiceAccountClient handles service account operations.
type ServiceAccountClient struct{ c *Client }

// List returns all service accounts.
func (sac *ServiceAccountClient) List(ctx context.Context) ([]ServiceAccount, error) {
	var resp struct {
		ServiceAccounts []ServiceAccount `json:"service_accounts"`
	}
	if err := sac.c.do(ctx, http.MethodGet, "/v1/service-accounts", nil, &resp); err != nil {
		return nil, err
	}
	return resp.ServiceAccounts, nil
}

// Create creates a new service account and returns the one-time secret.
func (sac *ServiceAccountClient) Create(ctx context.Context, req *CreateServiceAccountRequest) (*CreateServiceAccountResponse, error) {
	var resp CreateServiceAccountResponse
	if err := sac.c.do(ctx, http.MethodPost, "/v1/service-accounts", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete deletes a service account.
func (sac *ServiceAccountClient) Delete(ctx context.Context, id string) error {
	return sac.c.do(ctx, http.MethodDelete, "/v1/service-accounts/"+id, nil, nil)
}

// RotateKey generates new credentials for an existing service account.
func (sac *ServiceAccountClient) RotateKey(ctx context.Context, id string) (*CreateServiceAccountResponse, error) {
	var resp CreateServiceAccountResponse
	path := fmt.Sprintf("/v1/service-accounts/%s/rotate", id)
	if err := sac.c.do(ctx, http.MethodPost, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
