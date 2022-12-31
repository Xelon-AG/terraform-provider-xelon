package xelon

import (
	"context"
	"strconv"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceXelonCloud() *schema.Resource {
	return &schema.Resource{
		Description: "The cloud data source provides information about organization's cloud.",

		ReadContext: dataSourceXelonCloudRead,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"cloud_id": {
				Description: "The ID of the cloud.",
				Computed:    true,
				Type:        schema.TypeInt,
			},
			"name": {
				Description: "The name of the organization's cloud.",
				Required:    true,
				Type:        schema.TypeString,
			},
			"type": {
				Description: "The cloud type. `1` - private, `2` - public, `3` - whitelist.",
				Computed:    true,
				Type:        schema.TypeInt,
			},
		},
	}
}

func dataSourceXelonCloudRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	tenant, _, err := client.Tenants.GetCurrent(ctx)
	if err != nil {
		return diag.Errorf("getting tenant: %s", err)
	}

	clouds, _, err := client.Clouds.List(ctx, tenant.TenantID)
	if err != nil {
		return diag.Errorf("getting organization clouds: %s", err)
	}

	cloudName := d.Get("name").(string)
	for _, cloud := range clouds {
		if cloud.Name == cloudName {
			d.SetId(strconv.Itoa(cloud.ID))
			_ = d.Set("cloud_id", cloud.ID)
			_ = d.Set("name", cloud.Name)
			_ = d.Set("type", cloud.Type)
			return nil
		}
	}

	return diag.Errorf("No cloud found with name: %s", cloudName)
}
