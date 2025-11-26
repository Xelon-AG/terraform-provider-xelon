package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ datasource.DataSource              = (*tenantDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*tenantDataSource)(nil)
)

// tenantDataSource is the tenant datasource implementation.
type tenantDataSource struct {
	client *xelon.Client
}

// tenantDataSourceModel maps the tenant datasource schema data.
type tenantDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	ParentTenantID types.String `tfsdk:"parent_tenant_id"`
	Status         types.String `tfsdk:"status"`
	Type           types.String `tfsdk:"type"`
}

func NewTenantDataSource() datasource.DataSource {
	return &tenantDataSource{}
}

func (d *tenantDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_tenant"
}

func (d *tenantDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The tenant data source provides information about an existing tenant.

Tenants are the top-level entities in the Xelon Cloud. They are used
to group resources and manage access.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the tenant.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The tenant name.",
				Computed:            true,
				Optional:            true,
			},
			"parent_tenant_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the parent tenant.",
				Computed:            true,
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The status of the tenant.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the tenant (`Reseller` or `End Customer`).",
				Computed:            true,
			},
		},
	}
}

func (d *tenantDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*xelon.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unconfigured Xelon client",
			"Please report this issue to the provider developers.",
		)
		return
	}

	d.client = client
}

func (d *tenantDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data tenantDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tenantID := data.ID.ValueString()
	tenantName := data.Name.ValueString()

	// case to fetch current tenant
	if tenantID == "" && tenantName == "" {
		tflog.Info(ctx, "Searching for current tenant because ID and name are empty")

		tflog.Debug(ctx, "Getting current tenant")
		tenant, _, err := d.client.Tenants.GetCurrent(ctx)
		if err != nil {
			response.Diagnostics.AddError("Unable to get current tenant", err.Error())
			return
		}
		tflog.Debug(ctx, "Got current tenant", map[string]any{"data": tenant})

		// map response body to attributes
		data.ID = types.StringValue(tenant.ID)
		data.Name = types.StringValue(tenant.Name)
		data.ParentTenantID = types.StringValue(tenant.Parent)
		data.Status = types.StringValue(tenant.Status)
		data.Type = types.StringValue(tenant.Type)
	}

	if tenantID != "" {
		tflog.Info(ctx, "Searching for tenant by ID", map[string]any{"tenant_id": tenantID})

		tflog.Debug(ctx, "Getting tenant", map[string]any{"tenant_id": tenantID})
		tenant, resp, err := d.client.Tenants.Get(ctx, tenantID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
			response.Diagnostics.AddError("Unable to get tenant", err.Error())
			return
		}
		tflog.Debug(ctx, "Got tenant", map[string]any{"data": tenant})

		// if name is defined check that it's equal
		if tenantName != "" && tenantName != tenant.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual tenant name are different: expected '%s', got '%s'.", tenantName, tenant.Name),
			)
			return
		}

		// map response body to attributes
		data.ID = types.StringValue(tenant.ID)
		data.Name = types.StringValue(tenant.Name)
		data.ParentTenantID = types.StringValue(tenant.Parent)
		data.Status = types.StringValue(tenant.Status)
		data.Type = types.StringValue(tenant.Type)
	} else if tenantName != "" {
		tflog.Info(ctx, "Searching for tenant by name", map[string]any{"tenant_name": tenantName})

		tflog.Debug(ctx, "Getting tenants", map[string]any{"tenant_name": tenantName})
		tenants, _, err := d.client.Tenants.List(ctx, &xelon.TenantListOptions{Search: tenantName})
		if err != nil {
			response.Diagnostics.AddError("Unable to search tenants by name", err.Error())
			return
		}
		tflog.Debug(ctx, "Got tenants", map[string]any{"data": tenants})

		if len(tenants) == 0 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
		if len(tenants) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v tenants.", len(tenants)),
			)
			return
		}

		// map response body to attributes
		tenant := &tenants[0]
		data.ID = types.StringValue(tenant.ID)
		data.Name = types.StringValue(tenant.Name)
		data.ParentTenantID = types.StringValue(tenant.Parent)
		data.Status = types.StringValue(tenant.Status)
		data.Type = types.StringValue(tenant.Type)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
