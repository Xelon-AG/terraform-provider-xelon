package xelon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// New returns a schema.Provider for Xelon.
func New(version string) func() *schema.Provider {
	provider := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("XELON_BASE_URL", "https://hq.xelon.ch/api/v2/"),
				Description: "The base URL endpoint for Xelon HQ. Default is `https://hq.xelon.ch/api/v2/`. " +
					"Alternatively, can be configured using the `XELON_BASE_URL` environment variable.",
			},
			"client_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("XELON_CLIENT_ID", nil),
				Description: "The client ID for IP ranges. Alternatively, can be configured " +
					"using the `XELON_CLIENT_ID` environment variable.",
			},
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("XELON_TOKEN", nil),
				Description: "The Xelon access token. Alternatively, can be configured " +
					"using the `XELON_TOKEN` environment variable.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			// "xelon_cloud":  dataSourceXelonCloud(),
			// "xelon_tenant": dataSourceXelonTenant(),
		},

		ResourcesMap: map[string]*schema.Resource{
			// "xelon_device":             resourceXelonDevice(),
			// "xelon_persistent_storage": resourceXelonPersistentStorage(),
			// "xelon_network":            resourceXelonNetwork(),
			// "xelon_ssh_key":            resourceXelonSSHKey(),
		},
	}

	provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, version)
	}

	return func() *schema.Provider {
		return provider
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, version string) (interface{}, diag.Diagnostics) {
	config := &Config{
		BaseURL:         d.Get("base_url").(string),
		ClientID:        d.Get("client_id").(string),
		Token:           d.Get("token").(string),
		ProviderVersion: version,
	}

	client, err := config.Client(ctx)
	if err != nil {
		return nil, diag.Errorf("invalid provider config: %s", err)
	}

	return client, nil
}
