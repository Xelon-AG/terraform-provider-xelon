package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*deviceResource)(nil)
	_ resource.ResourceWithConfigure   = (*deviceResource)(nil)
	_ resource.ResourceWithImportState = (*deviceResource)(nil)
)

// deviceResource is the device resource implementation.
type deviceResource struct {
	client *xelon.Client
}

// deviceResourceModel maps the tag resource schema data.
type deviceResourceModel struct {
	CPUCoreCount types.Int64                  `tfsdk:"cpu_core_count"`
	DiskSize     types.Int64                  `tfsdk:"disk_size"`
	DisplayName  types.String                 `tfsdk:"display_name"`
	Hostname     types.String                 `tfsdk:"hostname"`
	ID           types.String                 `tfsdk:"id"`
	Memory       types.Int64                  `tfsdk:"memory"`
	Networks     []deviceNetworkResourceModel `tfsdk:"networks"`
	Password     types.String                 `tfsdk:"password"`
	SwapDiskSize types.Int64                  `tfsdk:"swap_disk_size"`
	TemplateID   types.String                 `tfsdk:"template_id"`
	TenantID     types.String                 `tfsdk:"tenant_id"`
}

type deviceNetworkResourceModel struct {
	Connected        types.Bool   `tfsdk:"connected"`
	ID               types.String `tfsdk:"id"`
	IPAddress        types.String `tfsdk:"ipv4_address"`
	NICControllerKey types.Int64  `tfsdk:"nic_controller_key"`
	NICKey           types.Int64  `tfsdk:"nic_key"`
	NICUnitNumber    types.Int64  `tfsdk:"nic_unit_number"`
}

func NewDeviceResource() resource.Resource {
	return &deviceResource{}
}

func (r *deviceResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_device"
}

func (r *deviceResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The device resource allows you to manage Xelon devices.

Devices are the virtual machines that run your applications.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"cpu_core_count": schema.Int64Attribute{
				MarkdownDescription: "The number of CPU cores to allocate to the device.",
				Required:            true,
			},
			"disk_size": schema.Int64Attribute{
				MarkdownDescription: "The size of the primary disk in GB.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The name of the device.",
				Required:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the device",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the device.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"memory": schema.Int64Attribute{
				MarkdownDescription: "The amount of RAM in GB to allocate to the device.",
				Required:            true,
			},
			"networks": schema.SetNestedAttribute{
				MarkdownDescription: "The networks configured for the device.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"connected": schema.BoolAttribute{
							Required: true,
						},
						"id": schema.StringAttribute{
							Required: true,
						},
						"ipv4_address": schema.StringAttribute{
							Required: true,
						},
						"nic_controller_key": schema.Int64Attribute{
							Required: true,
						},
						"nic_key": schema.Int64Attribute{
							Required: true,
						},
						"nic_unit_number": schema.Int64Attribute{
							Required: true,
						},
					},
				},
				Required: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for the device root or administrator user.",
				Required:            true,
				Sensitive:           true,
			},
			"swap_disk_size": schema.Int64Attribute{
				MarkdownDescription: "The size of the swap disk in GB.",
				Required:            true,
			},
			"template_id": schema.StringAttribute{
				MarkdownDescription: "The template ID used to create the device.",
				Required:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the device belongs.",
				Required:            true,
			},
		},
	}
}

func (r *deviceResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *deviceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data deviceResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	var networks []xelon.DeviceCreateNetwork
	for _, network := range data.Networks {
		networks = append(networks, xelon.DeviceCreateNetwork{
			ConnectOnPowerOn: network.Connected.ValueBool(),
			ControllerKey:    int(network.NICControllerKey.ValueInt64()),
			IPAddress:        network.IPAddress.ValueString(),
			Key:              int(network.NICKey.ValueInt64()),
			NetworkID:        network.ID.ValueString(),
			UnitNumber:       int(network.NICUnitNumber.ValueInt64()),
		})
	}

	createRequest := &xelon.DeviceCreateRequest{
		CPUCores:             int(data.CPUCoreCount.ValueInt64()),
		DiskSize:             int(data.DiskSize.ValueInt64()),
		DisplayName:          data.DisplayName.ValueString(),
		HostName:             data.Hostname.ValueString(),
		Networks:             networks,
		Password:             data.Password.ValueString(),
		PasswordConfirmation: data.Password.ValueString(),
		RAM:                  int(data.Memory.ValueInt64()),
		SwapDiskSize:         int(data.SwapDiskSize.ValueInt64()),
		TemplateID:           data.TemplateID.ValueString(),
		TenantID:             data.TenantID.ValueString(),
	}
	tflog.Debug(ctx, "Creating device", map[string]any{"payload": createRequest})
	createdDevice, _, err := r.client.Devices.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create device", err.Error())
		return
	}
	tflog.Debug(ctx, "Created device", map[string]any{"data": createdDevice})

	deviceID := createdDevice.ID
	tflog.Info(ctx, "Waiting for device to be powered on", map[string]any{"device_id": deviceID})
	err = helper.WaitPowerStateOn(ctx, r.client, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Device is not powered on", err.Error())
		return
	}
	tflog.Info(ctx, "Device is powered on", map[string]any{"device_id": deviceID})

	tflog.Debug(ctx, "Getting device with enriched properties", map[string]any{"device_id": deviceID})
	device, _, err := r.client.Devices.Get(ctx, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get device", err.Error())
		return
	}

	// map response body to attributes
	data.CPUCoreCount = types.Int64Value(int64(device.CPUCores))
	data.ID = types.StringValue(device.ID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *deviceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	panic("implement me")
}

func (r *deviceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	panic("implement me")
}

func (r *deviceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data deviceResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	deviceID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting device", map[string]any{"device_id": deviceID})
	_, err := r.client.Devices.Delete(ctx, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete device", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted device", map[string]any{"device_id": deviceID})
}

func (r *deviceResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
