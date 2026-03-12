// Package main is the entry point for the terraform-provider-vcstack binary.
package main

import (
	"github.com/Veritas-Calculus/vc-stack/terraform-provider-vcstack/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return provider.New()
		},
	})
}
