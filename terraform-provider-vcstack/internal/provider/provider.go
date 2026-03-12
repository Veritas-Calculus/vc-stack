// Package provider implements the VC Stack Terraform provider.
package provider

import (
	"context"

	"github.com/Veritas-Calculus/vc-stack/sdk/vcstack"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// New returns a new VC Stack Terraform provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCSTACK_ENDPOINT", nil),
				Description: "API endpoint URL (e.g. https://vc.example.com/api)",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("VCSTACK_TOKEN", nil),
				Description: "JWT Bearer token (alternative to API key auth)",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("VCSTACK_ACCESS_KEY_ID", nil),
				Description: "Service account Access Key ID (e.g. VC-AKIA-...)",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("VCSTACK_SECRET_KEY", nil),
				Description: "Service account Secret Key for HMAC-SHA256 signing",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"vcstack_instance":       resourceInstance(),
			"vcstack_network":        resourceNetwork(),
			"vcstack_subnet":         resourceSubnet(),
			"vcstack_volume":         resourceVolume(),
			"vcstack_security_group": resourceSecurityGroup(),
			"vcstack_floating_ip":    resourceFloatingIP(),
			"vcstack_ssh_key":        resourceSSHKey(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"vcstack_flavor": dataSourceFlavor(),
			"vcstack_image":  dataSourceImage(),
		},
		ConfigureContextFunc: configure,
	}
}

func configure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	endpoint := d.Get("endpoint").(string)
	client := vcstack.NewClient(endpoint)

	// API Key auth takes precedence.
	akid := d.Get("access_key_id").(string)
	sk := d.Get("secret_key").(string)
	if akid != "" && sk != "" {
		client.SetAPIKey(akid, sk)
	} else if token := d.Get("token").(string); token != "" {
		client.SetToken(token)
	} else {
		return nil, diag.Errorf("either token or access_key_id+secret_key must be provided")
	}

	return client, nil
}
