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
	_ datasource.DataSource              = (*loadBalancerDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*loadBalancerDataSource)(nil)
)

// loadBalancerDataSource is the load balancer datasource implementation.
type loadBalancerDataSource struct {
	client *xelon.Client
}

// loadBalancerDataSourceModel maps the load balancer datasource schema data.
type loadBalancerDataSourceModel struct {
	CloudID           types.String `tfsdk:"cloud_id"`
	DeviceIDs         types.Set    `tfsdk:"device_ids"` // []types.String
	ExternalIPAddress types.String `tfsdk:"external_ipv4_address"`
	ID                types.String `tfsdk:"id"`
	InternalIPAddress types.String `tfsdk:"internal_ipv4_address"`
	Name              types.String `tfsdk:"name"`
	TenantID          types.String `tfsdk:"tenant_id"`
	Type              types.String `tfsdk:"type"`
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
The load balancer data source provides information about an existing Xelon load balancers.

Load balancers sit in front of your application and distribute incoming traffic across multiple Devices.
`,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud associated with the load balancer.",
				Computed:            true,
			},
			"device_ids": schema.SetAttribute{
				MarkdownDescription: "The list of device IDs to associate with the load balancer.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"external_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The external IP address of the load balancer.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the load balancer.",
				Computed:            true,
				Optional:            true,
			},
			"internal_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The internal IP address of the load balancer.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The load balancer name.",
				Computed:            true,
				Optional:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the load balancer belongs.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The load balancing type (`layer4` or `layer7`).",
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

	loadBalancerID := data.ID.ValueString()
	loadBalancerName := data.Name.ValueString()
	if loadBalancerID == "" && loadBalancerName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	if loadBalancerID != "" {
		tflog.Info(ctx, "Searching for load balancer by ID", map[string]any{"load_balancer_id": loadBalancerID})

		tflog.Debug(ctx, "Getting load balancer", map[string]any{"load_balancer_id": loadBalancerID})
		loadBalancer, resp, err := d.client.LoadBalancers.Get(ctx, loadBalancerID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
			response.Diagnostics.AddError("Unable to get load balancer", err.Error())
			return
		}
		tflog.Debug(ctx, "Got load balancer", map[string]any{"data": loadBalancer})

		// if name is defined check that it's equal
		if loadBalancerName != "" && loadBalancerName != loadBalancer.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual load balancer name are different: expected '%s', got '%s'.", loadBalancerName, loadBalancer.Name),
			)
			return
		}

		// map response body to attributes
		deviceIDs := make([]string, 0, len(loadBalancer.AssignedDevices))
		for _, device := range loadBalancer.AssignedDevices {
			deviceIDs = append(deviceIDs, device.ID)
		}
		data.CloudID = types.StringValue(loadBalancer.Cloud.ID)
		data.DeviceIDs, diags = types.SetValueFrom(ctx, types.StringType, deviceIDs)
		response.Diagnostics.Append(diags...)
		data.ExternalIPAddress = types.StringValue(loadBalancer.ExternalIPAddress)
		data.ID = types.StringValue(loadBalancer.ID)
		data.InternalIPAddress = types.StringValue(loadBalancer.InternalIPAddress)
		data.Name = types.StringValue(loadBalancer.Name)
		data.TenantID = types.StringValue(loadBalancer.Tenant.ID)
		data.Type = types.StringValue(loadBalancer.Type)
	} else {
		tflog.Info(ctx, "Searching for load balancer by name", map[string]any{"load_balancer_name": loadBalancerName})

		tflog.Info(ctx, "Getting load balancers", map[string]any{"load_balancer_name": loadBalancerName})
		loadBalancers, _, err := d.client.LoadBalancers.List(ctx, &xelon.LoadBalancerListOptions{Search: loadBalancerName})
		if err != nil {
			response.Diagnostics.AddError("Unable to search load balancers by name", err.Error())
			return
		}
		tflog.Debug(ctx, "Got load balancers", map[string]any{"data": loadBalancers})

		if len(loadBalancers) == 0 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
		if len(loadBalancers) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v load balancers.", len(loadBalancers)),
			)
			return
		}

		loadBalancer := &loadBalancers[0]
		// enrich data because not all fields are exposed via list API
		tflog.Debug(ctx, "Getting load balancer", map[string]any{"load_balancer_id": loadBalancer.ID})
		loadBalancer, _, err = d.client.LoadBalancers.Get(ctx, loadBalancer.ID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get load balancer", err.Error())
			return
		}
		tflog.Debug(ctx, "Got load balancer", map[string]any{"data": loadBalancer})

		// map response body to attributes
		deviceIDs := make([]string, 0, len(loadBalancer.AssignedDevices))
		for _, device := range loadBalancer.AssignedDevices {
			deviceIDs = append(deviceIDs, device.ID)
		}
		data.CloudID = types.StringValue(loadBalancer.Cloud.ID)
		data.DeviceIDs, diags = types.SetValueFrom(ctx, types.StringType, deviceIDs)
		response.Diagnostics.Append(diags...)
		data.ExternalIPAddress = types.StringValue(loadBalancer.ExternalIPAddress)
		data.ID = types.StringValue(loadBalancer.ID)
		data.InternalIPAddress = types.StringValue(loadBalancer.InternalIPAddress)
		data.Name = types.StringValue(loadBalancer.Name)
		data.TenantID = types.StringValue(loadBalancer.Tenant.ID)
		data.Type = types.StringValue(loadBalancer.Type)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
