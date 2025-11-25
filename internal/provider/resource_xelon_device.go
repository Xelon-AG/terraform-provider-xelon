package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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

// deviceResourceModel maps the device resource schema data.
type deviceResourceModel struct {
	BackupJobID      types.Int64                  `tfsdk:"backup_job_id"`
	CPUCoreCount     types.Int64                  `tfsdk:"cpu_core_count"`
	DiskSize         types.Int64                  `tfsdk:"disk_size"`
	DisplayName      types.String                 `tfsdk:"display_name"`
	EnableMonitoring types.Bool                   `tfsdk:"enable_monitoring"`
	Hostname         types.String                 `tfsdk:"hostname"`
	ID               types.String                 `tfsdk:"id"`
	Memory           types.Int64                  `tfsdk:"memory"`
	Networks         []deviceNetworkResourceModel `tfsdk:"networks"`
	Password         types.String                 `tfsdk:"password"`
	SendEmail        types.Bool                   `tfsdk:"send_email"`
	SSHKeyID         types.String                 `tfsdk:"ssh_key_id"`
	ScriptID         types.String                 `tfsdk:"script_id"`
	SwapDiskSize     types.Int64                  `tfsdk:"swap_disk_size"`
	TemplateID       types.String                 `tfsdk:"template_id"`
	TenantID         types.String                 `tfsdk:"tenant_id"`
}

type deviceNetworkResourceModel struct {
	Connected   types.Bool   `tfsdk:"connected"`
	ID          types.String `tfsdk:"id"`
	IPAddress   types.String `tfsdk:"ipv4_address"`
	IPAddressID types.String `tfsdk:"ipv4_address_id"`
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
			"backup_job_id": schema.Int64Attribute{
				MarkdownDescription: "The ID for the backup job.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
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
			"enable_monitoring": schema.BoolAttribute{
				MarkdownDescription: "Whether to enable monitoring for the device.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the device.",
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
							MarkdownDescription: "Whether the network should automatically connect when the device powers on.",
							Required:            true,
						},
						"id": schema.StringAttribute{
							MarkdownDescription: "The network ID to which the device will connect.",
							Required:            true,
						},
						"ipv4_address": schema.StringAttribute{
							MarkdownDescription: "The static IP address for the network connection.",
							Optional:            true,
						},
						"ipv4_address_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the static IP address for the network connection.",
							Optional:            true,
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
			"send_email": schema.BoolAttribute{
				MarkdownDescription: "Whether to send an email notification upon successful device creation.",
				Optional:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"script_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the script to be executed during the device setup.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_key_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the SSH key to be used for authentication.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"swap_disk_size": schema.Int64Attribute{
				MarkdownDescription: "The size of the swap disk in GB.",
				Required:            true,
			},
			"template_id": schema.StringAttribute{
				MarkdownDescription: "The template ID used to create the device.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
		n := xelon.DeviceCreateNetwork{
			ConnectOnPowerOn: network.Connected.ValueBool(),
			NetworkID:        network.ID.ValueString(),
		}
		if network.IPAddress.ValueString() != "" {
			n.IPAddress = network.IPAddress.ValueString()
		}
		if network.IPAddressID.ValueString() != "" {
			n.IPAddress = network.IPAddressID.ValueString()
		}
		networks = append(networks, n)
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
	err = helper.WaitDevicePowerStateOn(ctx, r.client, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for device to be powered on", err.Error())
		return
	}
	tflog.Info(ctx, "Device is powered on", map[string]any{"device_id": deviceID})

	tflog.Info(ctx, "Waiting for device to be ready", map[string]any{"device_id": deviceID})
	err = helper.WaitDeviceStateReady(ctx, r.client, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for device to be ready", err.Error())
		return
	}
	tflog.Info(ctx, "Device is ready", map[string]any{"device_id": deviceID})

	tflog.Debug(ctx, "Getting device with enriched properties", map[string]any{"device_id": deviceID})
	device, _, err := r.client.Devices.Get(ctx, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get device", err.Error())
		return
	}
	tflog.Debug(ctx, "Got device with enriched properties", map[string]any{"data": device})

	// map response body to attributes
	data.CPUCoreCount = types.Int64Value(int64(device.CPUCores))
	data.DisplayName = types.StringValue(device.DisplayName)
	data.EnableMonitoring = types.BoolValue(device.MonitoringEnabled)
	data.Hostname = types.StringValue(device.HostName)
	data.ID = types.StringValue(device.ID)
	data.Memory = types.Int64Value(int64(device.RAM))

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *deviceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data deviceResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	deviceID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting device", map[string]any{"device_id": deviceID})
	device, resp, err := r.client.Devices.Get(ctx, deviceID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the tag is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get device", err.Error())
		return
	}
	tflog.Debug(ctx, "Got device", map[string]any{"data": device})

	// map response body to attributes
	data.CPUCoreCount = types.Int64Value(int64(device.CPUCores))
	data.DisplayName = types.StringValue(device.DisplayName)
	data.EnableMonitoring = types.BoolValue(device.MonitoringEnabled)
	data.Hostname = types.StringValue(device.HostName)
	data.ID = types.StringValue(device.ID)
	data.Memory = types.Int64Value(int64(device.RAM))

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *deviceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	// var plan, state deviceResourceModel
	//
	// // read plan and state data into the model
	// response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	// response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	// if response.Diagnostics.HasError() {
	// 	return
	// }
	//
	// deviceID := state.ID.ValueString()
	//
	// if !plan.DisplayName.Equal(state.DisplayName) {
	// 	updateRequest := &xelon.DeviceUpdateRequest{
	// 		DisplayName: plan.DisplayName.ValueString(),
	// 	}
	// 	tflog.Debug(ctx, "Updating device name", map[string]any{"device_id": deviceID, "payload": updateRequest})
	// 	updatedDevice, _, err := r.client.Devices.Update(ctx, deviceID, updateRequest)
	// 	if err != nil {
	// 		response.Diagnostics.AddError("Unable to update device name", err.Error())
	// 		return
	// 	}
	// 	tflog.Debug(ctx, "Updated device name", map[string]any{"data": updatedDevice})
	//
	// 	plan.DisplayName = types.StringValue(updatedDevice.DisplayName)
	// }
	//
	// if !plan.CPUCoreCount.Equal(state.CPUCoreCount) || !plan.Memory.Equal(state.Memory) {
	// 	// device must be stopped before changing CPU count and RAM
	// 	tflog.Debug(ctx, "Getting device", map[string]any{"device_id": deviceID})
	// 	device, _, err := r.client.Devices.Get(ctx, deviceID)
	// 	if err != nil {
	// 		response.Diagnostics.AddError("Unable to get device", err.Error())
	// 		return
	// 	}
	// 	tflog.Debug(ctx, "Got device", map[string]any{"data": device})
	// 	if device.PoweredOn {
	// 		tflog.Info(ctx, "Stopping device", map[string]any{"device_id": deviceID})
	// 		_, err := r.client.Devices.Stop(ctx, deviceID)
	// 		if err != nil {
	// 			response.Diagnostics.AddError("Unable to stop device", err.Error())
	// 			return
	// 		}
	//
	// 		err = helper.WaitDevicePowerStateOff(ctx, r.client, deviceID)
	// 		if err != nil {
	// 			response.Diagnostics.AddError("Unable to wait for device to be powered off", err.Error())
	// 			return
	// 		}
	// 	}
	//
	// 	updateRequest := &xelon.DeviceUpdateHardwareRequest{
	// 		CPUCores: int(plan.CPUCoreCount.ValueInt64()),
	// 		RAM:      int(plan.Memory.ValueInt64()),
	// 	}
	// 	tflog.Debug(ctx, "Updating device hardware", map[string]any{"device_id": deviceID, "payload": updateRequest})
	// 	updatedDevice, _, err := r.client.Devices.UpdateHardware(ctx, deviceID, updateRequest)
	// 	if err != nil {
	// 		response.Diagnostics.AddError("Unable to update device hardware", err.Error())
	// 		return
	// 	}
	// 	tflog.Debug(ctx, "Updated device hardware", map[string]any{"data": updatedDevice})
	//
	// 	tflog.Debug(ctx, "Getting device with enriched data", map[string]any{"device_id": deviceID})
	// 	device, _, err = r.client.Devices.Get(ctx, deviceID)
	// 	if err != nil {
	// 		response.Diagnostics.AddError("Unable to get device", err.Error())
	// 		return
	// 	}
	// 	tflog.Debug(ctx, "Got device with enriched data", map[string]any{"data": device})
	// 	if !device.PoweredOn {
	// 		tflog.Info(ctx, "Starting device", map[string]any{"device_id": deviceID})
	// 		_, err := r.client.Devices.Start(ctx, deviceID)
	// 		if err != nil {
	// 			response.Diagnostics.AddError("Unable to start device", err.Error())
	// 			return
	// 		}
	//
	// 		err = helper.WaitDevicePowerStateOn(ctx, r.client, deviceID)
	// 		if err != nil {
	// 			response.Diagnostics.AddError("Unable to wait for device to be powered on", err.Error())
	// 			return
	// 		}
	// 	}
	//
	// 	err = helper.WaitDeviceHostnameConfigured(ctx, r.client, deviceID)
	// 	if err != nil {
	// 		response.Diagnostics.AddError("Unable to wait for device to have hostname configured", err.Error())
	// 		return
	// 	}
	//
	// 	tflog.Debug(ctx, "Getting device with enriched data", map[string]any{"device_id": deviceID})
	// 	device, _, err = r.client.Devices.Get(ctx, deviceID)
	// 	if err != nil {
	// 		response.Diagnostics.AddError("Unable to get device", err.Error())
	// 		return
	// 	}
	// 	tflog.Debug(ctx, "Got device with enriched data", map[string]any{"data": device})
	//
	// 	plan.CPUCoreCount = types.Int64Value(int64(device.CPUCores))
	// 	plan.Memory = types.Int64Value(int64(device.RAM))
	// }
	//
	// diags := response.State.Set(ctx, &plan)
	// response.Diagnostics.Append(diags...)
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
	tflog.Debug(ctx, "Getting device", map[string]any{"device_id": deviceID})
	device, resp, err := r.client.Devices.Get(ctx, deviceID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return
		}
		response.Diagnostics.AddError("Unable to get device", err.Error())
		return
	}
	tflog.Debug(ctx, "Got device", map[string]any{"data": device})

	if device.PoweredOn {
		tflog.Info(ctx, "Stopping device", map[string]any{"data": device})
		_, err := r.client.Devices.Stop(ctx, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to stop device", err.Error())
			return
		}

		err = helper.WaitDevicePowerStateOff(ctx, r.client, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to wait for device to be powered off", err.Error())
			return
		}
	}

	tflog.Debug(ctx, "Deleting device", map[string]any{"device_id": deviceID})
	_, err = r.client.Devices.Delete(ctx, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete device", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted device", map[string]any{"device_id": deviceID})
}

func (r *deviceResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
