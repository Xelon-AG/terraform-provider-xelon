package xelon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider returns a schema.Provider for Xelon.
func Provider() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("XELON_BASE_URL", "https://hq.xelon.ch/api/service/"),
				Description: "The base URL endpoint for Xelon HQ. Default is 'https://hq.xelon.ch/api/service/'.",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("XELON_TOKEN", nil),
				Description: "The Xelon access token.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"xelon_ssh_key": resourceXelonSSHKey(),
		},
	}

	provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := provider.TerraformVersion
		return providerConfigure(ctx, d, terraformVersion)
	}

	return provider
}

func providerConfigure(_ context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	config := &Config{
		BaseURL:          d.Get("base_url").(string),
		Token:            d.Get("token").(string),
		TerraformVersion: terraformVersion,
	}

	return config.Client(), nil
}
