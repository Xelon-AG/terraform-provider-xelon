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
	_ datasource.DataSource              = (*firewallDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*firewallDataSource)(nil)
)

// firewallDataSource is the firewall data source implementation.
type firewallDataSource struct {
	client *xelon.Client
}

// firewallDataSourceModel maps the firewall data source schema data.
type firewallDataSourceModel struct {
	CloudID           types.String `tfsdk:"cloud_id"`
	ExternalIPAddress types.String `tfsdk:"external_ipv4_address"`
	ID                types.String `tfsdk:"id"`
	InternalIPAddress types.String `tfsdk:"internal_ipv4_address"`
	Name              types.String `tfsdk:"name"`
	TenantID          types.String `tfsdk:"tenant_id"`
}

func NewFirewallDataSource() datasource.DataSource {
	return &firewallDataSource{}
}

func (d *firewallDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_firewall"
}

func (d *firewallDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The firewall data source provides information about an existing Xelon firewall.
`,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud associated with the firewall.",
				Computed:            true,
			},
			"external_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The external IP address of the firewall.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the firewall.",
				Computed:            true,
				Optional:            true,
			},
			"internal_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The internal IP address of the firewall.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The firewall name.",
				Computed:            true,
				Optional:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the firewall belongs.",
				Computed:            true,
			},
		},
	}
}

func (d *firewallDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *firewallDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data firewallDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallID := data.ID.ValueString()
	firewallName := data.Name.ValueString()
	if firewallID == "" && firewallName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	if firewallID != "" {
		tflog.Info(ctx, "Searching for firewall by ID", map[string]any{"firewall_id": firewallID})

		tflog.Debug(ctx, "Getting firewall", map[string]any{"firewall_id": firewallID})
		firewall, resp, err := d.client.Firewalls.Get(ctx, firewallID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
			response.Diagnostics.AddError("Unable to get firewall", err.Error())
			return
		}
		tflog.Debug(ctx, "Got firewall", map[string]any{"data": firewall})

		// if name is defined check that it's equal
		if firewallName != "" && firewallName != firewall.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual firewall name are different: expected '%s', got '%s'.", firewallName, firewall.Name),
			)
			return
		}

		// map response body to attributes
		data.CloudID = types.StringValue(firewall.Cloud.ID)
		data.ExternalIPAddress = types.StringValue(firewall.ExternalIPAddress)
		data.ID = types.StringValue(firewall.ID)
		data.InternalIPAddress = types.StringValue(firewall.InternalIPAddress)
		data.Name = types.StringValue(firewall.Name)
		data.TenantID = types.StringValue(firewall.Tenant.ID)
	} else {
		tflog.Info(ctx, "Searching for firewall by name", map[string]any{"firewall_name": firewallName})

		tflog.Debug(ctx, "Getting firewalls", map[string]any{"firewall_name": firewallName})
		firewalls, _, err := d.client.Firewalls.List(ctx, &xelon.FirewallListOptions{Search: firewallName})
		if err != nil {
			response.Diagnostics.AddError("Unable to search firewalls by name", err.Error())
			return
		}
		tflog.Debug(ctx, "Got firewalls", map[string]any{"data": firewalls})

		if len(firewalls) == 0 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
		if len(firewalls) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v firewalls.", len(firewalls)),
			)
			return
		}

		firewall := &firewalls[0]
		// enrich data because not all fields are exposed via list API
		tflog.Debug(ctx, "Getting firewall", map[string]any{"firewall_id": firewall.ID})
		firewall, _, err = d.client.Firewalls.Get(ctx, firewall.ID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get firewall", err.Error())
			return
		}
		tflog.Debug(ctx, "Got firewall", map[string]any{"data": firewall})

		// map response body to attributes
		data.CloudID = types.StringValue(firewall.Cloud.ID)
		data.ExternalIPAddress = types.StringValue(firewall.ExternalIPAddress)
		data.ID = types.StringValue(firewall.ID)
		data.InternalIPAddress = types.StringValue(firewall.InternalIPAddress)
		data.Name = types.StringValue(firewall.Name)
		data.TenantID = types.StringValue(firewall.Tenant.ID)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
