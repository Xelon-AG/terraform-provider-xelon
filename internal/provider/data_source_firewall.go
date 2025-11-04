package provider

import (
	"context"
	"fmt"

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

// firewallDataSource is the firewall datasource implementation.
type firewallDataSource struct {
	client *xelon.Client
}

// firewallDataSourceModel maps the firewall datasource schema data.
type firewallDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	CloudID             types.String `tfsdk:"cloud_id"`
	TenantID            types.String `tfsdk:"tenant_id"`
	InternalIPv4Address types.String `tfsdk:"internal_ipv4_address"`
	ExternalIPv4Address types.String `tfsdk:"external_ipv4_address"`
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

Firewalls provide network security and routing between internal and external networks.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the firewall.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The firewall name.",
				Computed:            true,
				Optional:            true,
			},
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The cloud ID where the firewall is deployed.",
				Computed:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID that owns the firewall.",
				Computed:            true,
			},
			"internal_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The internal IPv4 address.",
				Computed:            true,
			},
			"external_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The external IPv4 address.",
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

	var firewall *xelon.Firewall
	var err error

	// If ID is provided, get directly by ID
	if firewallID != "" {
		tflog.Debug(ctx, "Getting firewall by ID", map[string]any{"firewall_id": firewallID})
		firewall, _, err = d.client.Firewalls.Get(ctx, firewallID)
		if err != nil {
			response.Diagnostics.AddError(
				"Unable to get firewall by ID",
				fmt.Sprintf("Error: %s", err.Error()),
			)
			return
		}

		// If name is also provided, validate it matches
		if firewallName != "" && firewallName != firewall.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual firewall name are different: expected '%s', got '%s'.", firewallName, firewall.Name),
			)
			return
		}
	} else {
		// Search by name using server-side filtering
		opts := &xelon.FirewallListOptions{
			Sort:   "name",
			Search: firewallName,
		}

		tflog.Debug(ctx, "Getting firewalls with server-side search", map[string]any{"search": firewallName})
		firewalls, _, err := d.client.Firewalls.List(ctx, opts)
		if err != nil {
			response.Diagnostics.AddError("Unable to list firewalls", err.Error())
			return
		}
		tflog.Debug(ctx, "Got filtered firewalls from API", map[string]any{"count": len(firewalls)})

		// API search may return partial matches, find exact match
		for _, fw := range firewalls {
			if firewallName == fw.Name {
				firewall = &fw
				break
			}
		}

		if firewall == nil {
			response.Diagnostics.AddError(
				"No search results",
				"No firewall found matching the specified criteria. Please refine your search.",
			)
			return
		}
	}

	// map response body to attributes
	data.ID = types.StringValue(firewall.ID)
	data.Name = types.StringValue(firewall.Name)
	data.CloudID = types.StringValue(firewall.Cloud.ID)
	data.TenantID = types.StringValue(firewall.Tenant.ID)
	data.InternalIPv4Address = types.StringValue(firewall.InternalIPAddress)
	data.ExternalIPv4Address = types.StringValue(firewall.ExternalIPAddress)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
