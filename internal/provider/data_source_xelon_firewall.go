package provider

import (
	"context"
	"fmt"
	"iter"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
The firewall data source provides information about an existing Xelon firewall. Firewalls can be selected by ID or by an exact, case-sensitive name. Name lookup fails if zero or multiple firewalls match.
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
				MarkdownDescription: "The exact, case-sensitive firewall name. Name lookup fails if zero or multiple firewalls match.",
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

	var firewall *xelon.Firewall
	if firewallID != "" {
		firewall, diags = d.lookupByID(ctx, firewallID, firewallName)
	} else {
		firewall, diags = d.lookupByName(ctx, firewallName)
	}
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	data.CloudID = types.StringValue(firewall.Cloud.ID)
	data.ExternalIPAddress = types.StringValue(firewall.ExternalIPAddress)
	data.ID = types.StringValue(firewall.ID)
	data.InternalIPAddress = types.StringValue(firewall.InternalIPAddress)
	data.Name = types.StringValue(firewall.Name)
	data.TenantID = types.StringValue(firewall.Tenant.ID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (d *firewallDataSource) lookupByID(ctx context.Context, firewallID, expectedName string) (*xelon.Firewall, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	tflog.Debug(ctx, "reading firewall", map[string]any{"firewall_id": firewallID})

	tflog.Trace(ctx, "getting firewall via API", map[string]any{"firewall_id": firewallID})
	firewall, apiResponse, err := d.client.Firewalls.Get(ctx, firewallID)
	if err != nil {
		if apiResponse != nil && apiResponse.StatusCode == http.StatusNotFound {
			diagnostics.AddError(
				"Firewall not found",
				fmt.Sprintf("No firewall with ID %q was found.", firewallID),
			)
			return nil, diagnostics
		}
		diagnostics.AddError("Unable to read firewall", err.Error())
		return nil, diagnostics
	}
	tflog.Trace(ctx, "received firewall from API", map[string]any{
		"firewall_id": firewall.ID,
		"name":        firewall.Name,
	})

	if expectedName != "" && firewall.Name != expectedName {
		diagnostics.AddError(
			"Firewall name mismatch",
			fmt.Sprintf("Firewall %q has name %q, not configured name %q.", firewallID, firewall.Name, expectedName),
		)
		return nil, diagnostics
	}

	return firewall, diagnostics
}

func (d *firewallDataSource) lookupByName(ctx context.Context, firewallName string) (*xelon.Firewall, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	tflog.Debug(ctx, "reading firewall", map[string]any{"name": firewallName})

	tflog.Trace(ctx, "listing firewalls via API")
	firewalls, errf := d.client.Firewalls.All(ctx, nil)
	match, matchCount := findFirewallByName(firewalls, firewallName)
	if err := errf(); err != nil {
		diagnostics.AddError("Unable to read firewall", err.Error())
		return nil, diagnostics
	}

	switch matchCount {
	case 0:
		diagnostics.AddError(
			"Firewall not found",
			fmt.Sprintf("No firewall with name %q was found. Names are matched exactly and case-sensitively.", firewallName),
		)
		return nil, diagnostics
	case 1:
		// The list response does not expose all fields needed for state, so fetch full details.
		tflog.Trace(ctx, "getting firewall via API", map[string]any{"firewall_id": match.ID})
		firewall, _, err := d.client.Firewalls.Get(ctx, match.ID)
		if err != nil {
			diagnostics.AddError("Unable to read firewall", err.Error())
			return nil, diagnostics
		}
		tflog.Trace(ctx, "received firewall from API", map[string]any{
			"firewall_id": firewall.ID,
			"name":        firewall.Name,
		})

		return firewall, diagnostics
	default:
		diagnostics.AddError(
			"Ambiguous firewall",
			fmt.Sprintf("Found %d firewalls named %q. Use `id` to select a specific firewall.", matchCount, firewallName),
		)
		return nil, diagnostics
	}
}

func findFirewallByName(firewalls iter.Seq2[xelon.Firewall, *xelon.Response], name string) (*xelon.Firewall, int) {
	var match *xelon.Firewall
	matchCount := 0

	for firewall := range firewalls {
		if firewall.Name == name {
			match = &firewall
			matchCount++
		}
	}

	return match, matchCount
}
