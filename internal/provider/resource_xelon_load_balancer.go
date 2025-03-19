package provider

import (
	"context"
	"net/http"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*loadBalancerResource)(nil)
	_ resource.ResourceWithConfigure   = (*loadBalancerResource)(nil)
	_ resource.ResourceWithImportState = (*loadBalancerResource)(nil)
)

// loadBalancerResource is the load balancer resource implementation.
type loadBalancerResource struct {
	client *xelon.Client
}

// loadBalancerResourceModel maps the load balancer resource schema data.
type loadBalancerResourceModel struct {
	CloudID           types.String `tfsdk:"cloud_id"`
	DeviceIDs         types.Set    `tfsdk:"device_ids"` // []types.String
	ExternalIPAddress types.String `tfsdk:"external_ipv4_address"`
	ID                types.String `tfsdk:"id"`
	InternalIPAddress types.String `tfsdk:"internal_ipv4_address"`
	Name              types.String `tfsdk:"name"`
	NetworkID         types.String `tfsdk:"network_id"`
	TenantID          types.String `tfsdk:"tenant_id"`
	Type              types.String `tfsdk:"type"`
}

func NewLoadBalancerResource() resource.Resource {
	return &loadBalancerResource{}
}

func (r *loadBalancerResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_load_balancer"
}

func (r *loadBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The load balancer resource allows you to manage Xelon load balancers.

Load balancers sit in front of your application and distribute incoming traffic across multiple Devices.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud associated with the load balancer.",
				Required:            true,
			},
			"device_ids": schema.SetAttribute{
				MarkdownDescription: "The list of device IDs to associate with the load balancer.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
			},
			"external_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The external IP address of the load balancer.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the load balancer.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"internal_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The internal IP address of the load balancer. If not provided, an internal IP will be automatically assigned.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The load balancer name.",
				Required:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The network ID used to create the load balancer.",
				Required:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the load balancer belongs.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The load balancing type. Must be one of `layer4` or `layer7`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"layer4", "layer7"}...),
				},
			},
		},
	}
}

func (r *loadBalancerResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

	r.client = client
}

func (r *loadBalancerResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data loadBalancerResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &xelon.LoadBalancerCreateRequest{
		CloudID:           data.CloudID.ValueString(),
		InternalNetworkID: data.NetworkID.ValueString(),
		Type:              data.Type.ValueString(),
		Name:              data.Name.ValueString(),
		TenantID:          data.TenantID.ValueString(),
	}
	if data.InternalIPAddress.ValueString() != "" {
		createRequest.InternalIPAddress = data.InternalIPAddress.ValueString()
	}
	if len(data.DeviceIDs.Elements()) > 0 {
		deviceIDs := make([]types.String, 0, len(data.DeviceIDs.Elements()))
		diags = data.DeviceIDs.ElementsAs(ctx, &deviceIDs, false)
		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}
		var assignedDeviceIDs []string
		for _, deviceID := range deviceIDs {
			assignedDeviceIDs = append(assignedDeviceIDs, deviceID.ValueString())
		}
		createRequest.AssignedDeviceIDs = assignedDeviceIDs
	}
	tflog.Debug(ctx, "Creating load balancer", map[string]any{"payload": createRequest})
	createdLoadBalancer, _, err := r.client.LoadBalancers.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create load balancer", err.Error())
		return
	}
	tflog.Debug(ctx, "Created load balancer", map[string]any{"data": createdLoadBalancer})

	loadBalancerID := createdLoadBalancer.ID

	tflog.Info(ctx, "Waiting for load balancer to be ready", map[string]any{"load_balancer_id": loadBalancerID})
	err = helper.WaitLoadBalancerStateReady(ctx, r.client, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for load balancer to be ready", err.Error())
		return
	}
	tflog.Info(ctx, "Load balancer is ready", map[string]any{"load_balancer_id": loadBalancerID})

	tflog.Debug(ctx, "Getting load balancer with enriched properties", map[string]any{"load_balancer_id": loadBalancerID})
	loadBalancer, _, err := r.client.LoadBalancers.Get(ctx, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get load balancer", err.Error())
		return
	}
	tflog.Debug(ctx, "Got load balancer with enriched properties", map[string]any{"data": loadBalancer})

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

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *loadBalancerResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data loadBalancerResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting load balancer", map[string]any{"load_balancer_id": loadBalancerID})
	loadBalancer, resp, err := r.client.LoadBalancers.Get(ctx, loadBalancerID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the load balancer is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
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

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *loadBalancerResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state loadBalancerResourceModel

	// read plan and state data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := state.ID.ValueString()

	if !plan.Name.Equal(state.Name) {
		updateRequest := &xelon.LoadBalancerUpdateRequest{
			Name: plan.Name.ValueString(),
		}
		tflog.Debug(ctx, "Updating load balancer", map[string]any{"load_balancer_id": loadBalancerID, "payload": updateRequest})
		loadBalancer, _, err := r.client.LoadBalancers.Update(ctx, loadBalancerID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update load balancer", err.Error())
			return
		}
		tflog.Debug(ctx, "Updated load balancer", map[string]any{"load_balancer_id": loadBalancerID, "data": loadBalancer})

		tflog.Debug(ctx, "Getting load balancer with enriched data", map[string]any{"load_balancer_id": loadBalancerID})
		loadBalancer, _, err = r.client.LoadBalancers.Get(ctx, loadBalancerID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get load balancer", err.Error())
			return
		}
		tflog.Debug(ctx, "Got load balancer with enriched data", map[string]any{"data": loadBalancer})

		plan.Name = types.StringValue(loadBalancer.Name)
	}

	if !plan.DeviceIDs.Equal(state.DeviceIDs) {
		// get and sort device ids from plan for comparison later
		tfPlanDeviceIDs := make([]types.String, 0, len(plan.DeviceIDs.Elements()))
		diags := plan.DeviceIDs.ElementsAs(ctx, &tfPlanDeviceIDs, false)
		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}
		var planDeviceIDs []string
		for _, tfPlanDeviceID := range tfPlanDeviceIDs {
			planDeviceIDs = append(planDeviceIDs, tfPlanDeviceID.ValueString())
		}
		slices.Sort(planDeviceIDs)

		// get and sort device ids from state for comparison later
		tfStateDeviceIDs := make([]types.String, 0, len(state.DeviceIDs.Elements()))
		diags = state.DeviceIDs.ElementsAs(ctx, &tfStateDeviceIDs, false)
		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}
		var stateDeviceIDs []string
		for _, tfStateDeviceID := range tfStateDeviceIDs {
			stateDeviceIDs = append(stateDeviceIDs, tfStateDeviceID.ValueString())
		}
		slices.Sort(stateDeviceIDs)

		if !slices.Equal(planDeviceIDs, stateDeviceIDs) {
			// backend API cannot deal with nil, set empty slice
			if planDeviceIDs == nil {
				planDeviceIDs = []string{}
			}
			updateRequest := &xelon.LoadBalancerUpdateAssignedDevicesRequest{
				DeviceIDs: planDeviceIDs,
			}
			tflog.Debug(ctx, "Updating load balancer assigned devices", map[string]any{"load_balancer_id": loadBalancerID, "payload": updateRequest})
			_, err := r.client.LoadBalancers.UpdateAssignedDevices(ctx, loadBalancerID, updateRequest)
			if err != nil {
				response.Diagnostics.AddError("Unable to update load balancer", err.Error())
				return
			}
			tflog.Debug(ctx, "Updated load balancer assigned devices", map[string]any{"load_balancer_id": loadBalancerID})

			tflog.Debug(ctx, "Getting load balancer with enriched data", map[string]any{"load_balancer_id": loadBalancerID})
			loadBalancer, _, err := r.client.LoadBalancers.Get(ctx, loadBalancerID)
			if err != nil {
				response.Diagnostics.AddError("Unable to get load balancer", err.Error())
				return
			}
			tflog.Debug(ctx, "Got load balancer with enriched data", map[string]any{"data": loadBalancer})

			deviceIDs := make([]string, 0, len(loadBalancer.AssignedDevices))
			for _, device := range loadBalancer.AssignedDevices {
				deviceIDs = append(deviceIDs, device.ID)
			}
			plan.DeviceIDs, diags = types.SetValueFrom(ctx, types.StringType, deviceIDs)
			response.Diagnostics.Append(diags...)
		}
	}

	diags := response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}

func (r *loadBalancerResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data loadBalancerResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting load balancer", map[string]any{"load_balancer_id": loadBalancerID})
	_, err := r.client.LoadBalancers.Delete(ctx, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete load balancer", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted load balancer", map[string]any{"load_balancer_id": loadBalancerID})
}

func (r *loadBalancerResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
