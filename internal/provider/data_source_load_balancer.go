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
	_ datasource.DataSource              = (*loadBalancerDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*loadBalancerDataSource)(nil)
)

// loadBalancerDataSource is the load balancer datasource implementation.
type loadBalancerDataSource struct {
	client *xelon.Client
}

// loadBalancerDataSourceModel maps the load balancer datasource schema data.
type loadBalancerDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	CloudID             types.String `tfsdk:"cloud_id"`
	TenantID            types.String `tfsdk:"tenant_id"`
	InternalIPv4Address types.String `tfsdk:"internal_ipv4_address"`
	ExternalIPv4Address types.String `tfsdk:"external_ipv4_address"`
}

func NewLoadBalancerDataSource() datasource.DataSource {
	return &loadBalancerDataSource{}
}

func (d *loadBalancerDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_load_balancer"
}

func (d *loadBalancerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The load balancer data source provides information about an existing Xelon load balancer.

Load balancers distribute network traffic across multiple devices for high availability and scalability.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the load balancer.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The load balancer name.",
				Computed:            true,
				Optional:            true,
			},
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The cloud ID where the load balancer is deployed.",
				Computed:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID that owns the load balancer.",
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

func (d *loadBalancerDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *loadBalancerDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data loadBalancerDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	lbID := data.ID.ValueString()
	lbName := data.Name.ValueString()
	if lbID == "" && lbName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	var loadBalancer *xelon.LoadBalancer
	var err error

	// If ID is provided, get directly by ID
	if lbID != "" {
		tflog.Debug(ctx, "Getting load balancer by ID", map[string]any{"load_balancer_id": lbID})
		loadBalancer, _, err = d.client.LoadBalancers.Get(ctx, lbID)
		if err != nil {
			response.Diagnostics.AddError(
				"Unable to get load balancer by ID",
				fmt.Sprintf("Error: %s", err.Error()),
			)
			return
		}

		// If name is also provided, validate it matches
		if lbName != "" && lbName != loadBalancer.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual load balancer name are different: expected '%s', got '%s'.", lbName, loadBalancer.Name),
			)
			return
		}
	} else {
		// Search by name using server-side filtering
		opts := &xelon.LoadBalancerListOptions{
			Sort:   "name",
			Search: lbName,
		}

		tflog.Debug(ctx, "Getting load balancers with server-side search", map[string]any{"search": lbName})
		loadBalancers, _, err := d.client.LoadBalancers.List(ctx, opts)
		if err != nil {
			response.Diagnostics.AddError("Unable to list load balancers", err.Error())
			return
		}
		tflog.Debug(ctx, "Got filtered load balancers from API", map[string]any{"count": len(loadBalancers)})

		// API search may return partial matches, find exact match
		for _, lb := range loadBalancers {
			if lbName == lb.Name {
				loadBalancer = &lb
				break
			}
		}

		if loadBalancer == nil {
			response.Diagnostics.AddError(
				"No search results",
				"No load balancer found matching the specified criteria. Please refine your search.",
			)
			return
		}
	}

	// map response body to attributes
	data.ID = types.StringValue(loadBalancer.ID)
	data.Name = types.StringValue(loadBalancer.Name)
	data.CloudID = types.StringValue(loadBalancer.Cloud.ID)
	data.TenantID = types.StringValue(loadBalancer.Tenant.ID)
	data.InternalIPv4Address = types.StringValue(loadBalancer.InternalIPAddress)
	data.ExternalIPv4Address = types.StringValue(loadBalancer.ExternalIPAddress)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
