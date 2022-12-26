package xelon

import (
	"context"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceXelonTenant() *schema.Resource {
	return &schema.Resource{
		Description: "The tenant data source provides information about an existing tenant (organization).",

		ReadContext: dataSourceXelonTenantRead,

		SchemaVersion: 0,

		Schema: map[string]*schema.Schema{
			"active": {
				Description: "True if the organization is active.",
				Computed:    true,
				Type:        schema.TypeBool,
			},
			"name": {
				Description: "The name of the organization.",
				Computed:    true,
				Type:        schema.TypeString,
			},
			"tenant_id": {
				Description: "The ID of the organization.",
				Computed:    true,
				Type:        schema.TypeString,
			},
		},
	}
}

func dataSourceXelonTenantRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*xelon.Client)

	current, _, err := client.Tenants.GetCurrent(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	tenants, _, err := client.Tenants.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// workaround until API has a method to return single tenant info
	for _, tenant := range tenants {
		if tenant.ID == current.TenantID {
			current.Active = tenant.Active
			current.Name = tenant.Name
			break
		}
	}

	d.SetId(current.TenantID)
	_ = d.Set("active", current.Active)
	_ = d.Set("name", current.Name)
	_ = d.Set("tenant_id", current.TenantID)

	return nil
}
