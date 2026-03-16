package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Veritas-Calculus/vc-stack/sdk/vcstack"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a VC Stack compute instance (virtual machine).",
		CreateContext: resourceInstanceCreate,
		ReadContext:   resourceInstanceRead,
		DeleteContext: resourceInstanceDelete,
		Schema: map[string]*schema.Schema{
			"name":      {Type: schema.TypeString, Required: true, ForceNew: true, Description: "Instance name"},
			"flavor_id": {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "Flavor ID"},
			"image_id":  {Type: schema.TypeInt, Required: true, ForceNew: true, Description: "Image ID"},
			"network_id": {
				Type: schema.TypeInt, Optional: true, ForceNew: true,
				Description: "Network ID to attach the instance to",
			},
			"ssh_key_id": {
				Type: schema.TypeInt, Optional: true, ForceNew: true,
				Description: "SSH key ID for access",
			},
			"user_data": {
				Type: schema.TypeString, Optional: true, ForceNew: true,
				Description: "Cloud-init user data",
			},
			// Computed
			"uuid":        {Type: schema.TypeString, Computed: true},
			"status":      {Type: schema.TypeString, Computed: true},
			"ip_address":  {Type: schema.TypeString, Computed: true},
			"floating_ip": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceInstanceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	req := &vcstack.CreateInstanceRequest{
		Name:     d.Get("name").(string),
		FlavorID: d.Get("flavor_id").(int),
		ImageID:  d.Get("image_id").(int),
	}
	if v, ok := d.GetOk("network_id"); ok {
		req.NetworkID = v.(int)
	}
	if v, ok := d.GetOk("ssh_key_id"); ok {
		req.SSHKeyID = v.(int)
	}
	if v, ok := d.GetOk("user_data"); ok {
		req.UserData = v.(string)
	}

	instance, err := client.Instances.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(instance.ID))
	return resourceInstanceRead(ctx, d, meta)
}

func resourceInstanceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	instance, err := client.Instances.Get(ctx, d.Id())
	if err != nil {
		if apiErr, ok := err.(*vcstack.APIError); ok && apiErr.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("name", instance.Name)
	d.Set("uuid", instance.UUID)
	d.Set("status", instance.Status)
	d.Set("ip_address", instance.IPAddress)
	d.Set("floating_ip", instance.FloatingIP)
	return nil
}

func resourceInstanceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	if err := client.Instances.Delete(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// ──────────────────────────────────────────────────────────────────────

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a VC Stack virtual network.",
		CreateContext: resourceNetworkCreate,
		ReadContext:   resourceNetworkRead,
		DeleteContext: resourceNetworkDelete,
		Schema: map[string]*schema.Schema{
			"name":        {Type: schema.TypeString, Required: true, ForceNew: true},
			"description": {Type: schema.TypeString, Optional: true, ForceNew: true},
			"tenant_id":   {Type: schema.TypeString, Optional: true, ForceNew: true},
			"external":    {Type: schema.TypeBool, Optional: true, ForceNew: true, Default: false},
			"shared":      {Type: schema.TypeBool, Optional: true, ForceNew: true, Default: false},
			// Computed
			"network_type":    {Type: schema.TypeString, Computed: true},
			"segmentation_id": {Type: schema.TypeInt, Computed: true},
			"status":          {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	req := &vcstack.CreateNetworkRequest{
		Name:     d.Get("name").(string),
		External: d.Get("external").(bool),
		Shared:   d.Get("shared").(bool),
	}
	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}
	if v, ok := d.GetOk("tenant_id"); ok {
		req.TenantID = v.(string)
	}

	net, err := client.Networks.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(net.ID))
	return resourceNetworkRead(ctx, d, meta)
}

func resourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	net, err := client.Networks.Get(ctx, d.Id())
	if err != nil {
		if apiErr, ok := err.(*vcstack.APIError); ok && apiErr.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	d.Set("name", net.Name)
	d.Set("network_type", net.NetworkType)
	d.Set("segmentation_id", net.SegmentationID)
	d.Set("status", net.Status)
	return nil
}

func resourceNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	if err := client.Networks.Delete(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// ──────────────────────────────────────────────────────────────────────

func resourceSubnet() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a VC Stack subnet within a network.",
		CreateContext: resourceSubnetCreate,
		ReadContext:   resourceSubnetRead,
		DeleteContext: resourceSubnetDelete,
		Schema: map[string]*schema.Schema{
			"name":       {Type: schema.TypeString, Required: true, ForceNew: true},
			"cidr":       {Type: schema.TypeString, Required: true, ForceNew: true},
			"gateway":    {Type: schema.TypeString, Optional: true, ForceNew: true},
			"network_id": {Type: schema.TypeInt, Required: true, ForceNew: true},
			"tenant_id":  {Type: schema.TypeString, Optional: true, ForceNew: true},
		},
	}
}

func resourceSubnetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	req := &vcstack.CreateSubnetRequest{
		Name:      d.Get("name").(string),
		CIDR:      d.Get("cidr").(string),
		NetworkID: d.Get("network_id").(int),
	}
	if v, ok := d.GetOk("gateway"); ok {
		req.Gateway = v.(string)
	}
	if v, ok := d.GetOk("tenant_id"); ok {
		req.TenantID = v.(string)
	}

	subnet, err := client.Subnets.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(subnet.ID))
	return resourceSubnetRead(ctx, d, meta)
}

func resourceSubnetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Subnet read uses same data from create (no dedicated Get endpoint needed).
	return nil
}

func resourceSubnetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	if err := client.Subnets.Delete(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// ──────────────────────────────────────────────────────────────────────

func resourceVolume() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a VC Stack block storage volume.",
		CreateContext: resourceVolumeCreate,
		ReadContext:   resourceVolumeRead,
		DeleteContext: resourceVolumeDelete,
		Schema: map[string]*schema.Schema{
			"name":        {Type: schema.TypeString, Required: true, ForceNew: true},
			"size_gb":     {Type: schema.TypeInt, Required: true, ForceNew: true},
			"volume_type": {Type: schema.TypeString, Optional: true, ForceNew: true},
			// Computed
			"status": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceVolumeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	req := &vcstack.CreateVolumeRequest{
		Name:   d.Get("name").(string),
		SizeGB: d.Get("size_gb").(int),
	}
	if v, ok := d.GetOk("volume_type"); ok {
		req.VolumeType = v.(string)
	}

	vol, err := client.Volumes.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(vol.ID))
	return resourceVolumeRead(ctx, d, meta)
}

func resourceVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	vol, err := client.Volumes.Get(ctx, d.Id())
	if err != nil {
		if apiErr, ok := err.(*vcstack.APIError); ok && apiErr.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	d.Set("name", vol.Name)
	d.Set("size_gb", vol.SizeGB)
	d.Set("status", vol.Status)
	return nil
}

func resourceVolumeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	if err := client.Volumes.Delete(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// ──────────────────────────────────────────────────────────────────────

func resourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a VC Stack network security group.",
		CreateContext: resourceSecurityGroupCreate,
		ReadContext:   resourceSecurityGroupRead,
		DeleteContext: resourceSecurityGroupDelete,
		Schema: map[string]*schema.Schema{
			"name":        {Type: schema.TypeString, Required: true, ForceNew: true},
			"description": {Type: schema.TypeString, Optional: true, ForceNew: true},
			"tenant_id":   {Type: schema.TypeString, Optional: true, ForceNew: true},
		},
	}
}

func resourceSecurityGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	req := &vcstack.CreateSecurityGroupRequest{
		Name: d.Get("name").(string),
	}
	if v, ok := d.GetOk("description"); ok {
		req.Description = v.(string)
	}
	if v, ok := d.GetOk("tenant_id"); ok {
		req.TenantID = v.(string)
	}

	sg, err := client.SecurityGroups.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(sg.ID))
	return nil
}

func resourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	sg, err := client.SecurityGroups.Get(ctx, d.Id())
	if err != nil {
		if apiErr, ok := err.(*vcstack.APIError); ok && apiErr.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	d.Set("name", sg.Name)
	d.Set("description", sg.Description)
	return nil
}

func resourceSecurityGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	if err := client.SecurityGroups.Delete(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// ──────────────────────────────────────────────────────────────────────

func resourceFloatingIP() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a VC Stack floating IP address.",
		CreateContext: resourceFloatingIPCreate,
		ReadContext:   resourceFloatingIPRead,
		DeleteContext: resourceFloatingIPDelete,
		Schema: map[string]*schema.Schema{
			"network_id": {Type: schema.TypeInt, Required: true, ForceNew: true},
			"tenant_id":  {Type: schema.TypeString, Required: true, ForceNew: true},
			// Computed
			"address": {Type: schema.TypeString, Computed: true},
			"status":  {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceFloatingIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	fip, err := client.FloatingIPs.Create(ctx, d.Get("network_id").(int), d.Get("tenant_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(fip.ID))
	d.Set("address", fip.Address)
	d.Set("status", fip.Status)
	return nil
}

func resourceFloatingIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Floating IP does not have a dedicated Get endpoint; rely on List for refreshes.
	return nil
}

func resourceFloatingIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	if err := client.FloatingIPs.Delete(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// ──────────────────────────────────────────────────────────────────────

func resourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a VC Stack SSH key pair.",
		CreateContext: resourceSSHKeyCreate,
		ReadContext:   resourceSSHKeyRead,
		DeleteContext: resourceSSHKeyDelete,
		Schema: map[string]*schema.Schema{
			"name":       {Type: schema.TypeString, Required: true, ForceNew: true},
			"public_key": {Type: schema.TypeString, Required: true, ForceNew: true},
			// Computed
			"fingerprint": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceSSHKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	key, err := client.SSHKeys.Create(ctx, d.Get("name").(string), d.Get("public_key").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(key.ID))
	d.Set("fingerprint", key.Fingerprint)
	return nil
}

func resourceSSHKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceSSHKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	if err := client.SSHKeys.Delete(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Data Sources
// ──────────────────────────────────────────────────────────────────────

func dataSourceFlavor() *schema.Resource {
	return &schema.Resource{
		Description: "Look up a VC Stack flavor by name.",
		ReadContext: dataSourceFlavorRead,
		Schema: map[string]*schema.Schema{
			"name":  {Type: schema.TypeString, Required: true},
			"vcpus": {Type: schema.TypeInt, Computed: true},
			"ram":   {Type: schema.TypeInt, Computed: true, Description: "RAM in MB"},
			"disk":  {Type: schema.TypeInt, Computed: true, Description: "Disk in GB"},
		},
	}
}

func dataSourceFlavorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	flavors, err := client.Flavors.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	for _, f := range flavors {
		if f.Name == name {
			d.SetId(strconv.Itoa(f.ID))
			d.Set("vcpus", f.VCPUs)
			d.Set("ram", f.RAM)
			d.Set("disk", f.Disk)
			return nil
		}
	}

	return diag.Errorf("flavor %q not found", name)
}

func dataSourceImage() *schema.Resource {
	return &schema.Resource{
		Description: "Look up a VC Stack image by name.",
		ReadContext: dataSourceImageRead,
		Schema: map[string]*schema.Schema{
			"name":        {Type: schema.TypeString, Required: true},
			"status":      {Type: schema.TypeString, Computed: true},
			"disk_format": {Type: schema.TypeString, Computed: true},
			"min_disk":    {Type: schema.TypeInt, Computed: true, Description: "Minimum disk in GB"},
		},
	}
}

func dataSourceImageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	images, err := client.Images.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	for _, img := range images {
		if img.Name == name {
			d.SetId(strconv.Itoa(img.ID))
			d.Set("status", img.Status)
			d.Set("disk_format", img.DiskFormat)
			d.Set("min_disk", img.MinDisk)
			return nil
		}
	}

	return diag.Errorf("image %q not found", name)
}

// ──────────────────────────────────────────────────────────────────────
// Router Resource
// ──────────────────────────────────────────────────────────────────────

func resourceRouter() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a VC Stack virtual router.",
		CreateContext: resourceRouterCreate,
		ReadContext:   resourceRouterRead,
		DeleteContext: resourceRouterDelete,
		Schema: map[string]*schema.Schema{
			"name":                {Type: schema.TypeString, Required: true, ForceNew: true},
			"description":         {Type: schema.TypeString, Optional: true, ForceNew: true},
			"external_network_id": {Type: schema.TypeInt, Optional: true, ForceNew: true, Description: "ID of the external network for gateway"},
			"enable_snat":         {Type: schema.TypeBool, Optional: true, Default: false, ForceNew: true},
			// Computed
			"status":     {Type: schema.TypeString, Computed: true},
			"gateway_ip": {Type: schema.TypeString, Computed: true},
		},
	}
}

func resourceRouterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	req := &vcstack.CreateRouterRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		EnableSNAT:  d.Get("enable_snat").(bool),
	}
	if v, ok := d.GetOk("external_network_id"); ok {
		req.ExternalNetworkID = v.(int)
	}
	router, err := client.Routers.Create(ctx, req)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(router.ID))
	d.Set("status", router.Status)
	d.Set("gateway_ip", router.GatewayIP)
	return nil
}

func resourceRouterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	router, err := client.Routers.Get(ctx, d.Id())
	if err != nil {
		d.SetId("")
		return nil
	}
	d.Set("name", router.Name)
	d.Set("description", router.Description)
	d.Set("status", router.Status)
	d.Set("gateway_ip", router.GatewayIP)
	d.Set("enable_snat", router.EnableSNAT)
	if router.ExternalNetworkID != 0 {
		d.Set("external_network_id", router.ExternalNetworkID)
	}
	return nil
}

func resourceRouterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	if err := client.Routers.Delete(ctx, d.Id()); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

// ──────────────────────────────────────────────────────────────────────
// Network Data Source
// ──────────────────────────────────────────────────────────────────────

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Look up a VC Stack network by name.",
		ReadContext: dataSourceNetworkRead,
		Schema: map[string]*schema.Schema{
			"name":     {Type: schema.TypeString, Required: true},
			"external": {Type: schema.TypeBool, Computed: true},
			"shared":   {Type: schema.TypeBool, Computed: true},
			"status":   {Type: schema.TypeString, Computed: true},
		},
	}
}

func dataSourceNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*vcstack.Client)
	networks, err := client.Networks.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	name := d.Get("name").(string)
	for _, n := range networks {
		if n.Name == name {
			d.SetId(strconv.Itoa(n.ID))
			d.Set("external", n.External)
			d.Set("shared", n.Shared)
			d.Set("status", n.Status)
			return nil
		}
	}

	return diag.Errorf("network %q not found", name)
}

// Compile-time check to satisfy fmt.Stringer for unused import.
var _ = fmt.Sprint
