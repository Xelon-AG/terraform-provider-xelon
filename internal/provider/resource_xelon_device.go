package provider

import (
	"context"
	"errors"
	"net/http"
	"slices"

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
	_ resource.ResourceWithModifyPlan  = (*deviceResource)(nil)
)

// deviceResource is the device resource implementation.
type deviceResource struct {
	client *xelon.Client
}

// deviceResourceModel maps the device resource schema data.
type deviceResourceModel struct {
	BackupJobID      types.Int64                  `tfsdk:"backup_job_id"`
	CPUCoreCount     types.Int64                  `tfsdk:"cpu_core_count"`
	CPUCoreHotPlug   types.Bool                   `tfsdk:"cpu_core_hotplug"`
	DiskID           types.String                 `tfsdk:"disk_id"`
	DiskSize         types.Int64                  `tfsdk:"disk_size"`
	DisplayName      types.String                 `tfsdk:"display_name"`
	EnableMonitoring types.Bool                   `tfsdk:"enable_monitoring"`
	Hostname         types.String                 `tfsdk:"hostname"`
	ID               types.String                 `tfsdk:"id"`
	Memory           types.Int64                  `tfsdk:"memory"`
	MemoryHotPlug    types.Bool                   `tfsdk:"memory_hotplug"`
	Networks         []deviceNetworkResourceModel `tfsdk:"networks"`
	Password         types.String                 `tfsdk:"password"`
	SendEmail        types.Bool                   `tfsdk:"send_email"`
	SSHKeyID         types.String                 `tfsdk:"ssh_key_id"`
	ScriptID         types.String                 `tfsdk:"script_id"`
	SwapDiskID       types.String                 `tfsdk:"swap_disk_id"`
	SwapDiskSize     types.Int64                  `tfsdk:"swap_disk_size"`
	TemplateID       types.String                 `tfsdk:"template_id"`
	TenantID         types.String                 `tfsdk:"tenant_id"`
	UserData         types.String                 `tfsdk:"user_data"`
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
			"cpu_core_hotplug": schema.BoolAttribute{
				MarkdownDescription: "If `true`, enables CPU core hot‑plug functionality for the device. " +
					"It allows dynamically adding or removing CPU cores without powering off the device.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"disk_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the primary disk.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disk_size": schema.Int64Attribute{
				MarkdownDescription: "The size of the primary disk in GB.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					helper.ExpandOnlyStorageSizeModifier(),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The name of the device.",
				Required:            true,
			},
			"enable_monitoring": schema.BoolAttribute{
				DeprecationMessage:  "enable_monitoring is deprecated and will not be supported in a future release.",
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
			"memory_hotplug": schema.BoolAttribute{
				MarkdownDescription: "If `true`, enables memory hot‑plug functionality for the device. " +
					"It allows dynamically increasing or decreasing the amount of RAM without powering off the device.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"networks": schema.SetNestedAttribute{
				MarkdownDescription: "The networks configured for the device.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"connected": schema.BoolAttribute{
							MarkdownDescription: "Whether the network should automatically connect when the device powers on.",
							Optional:            true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"id": schema.StringAttribute{
							MarkdownDescription: "The network ID to which the device will connect.",
							Required:            true,
						},
						"ipv4_address": schema.StringAttribute{
							MarkdownDescription: "The IPv4 address assigned to the device on this network. " +
								"Specify a value for a static address; when omitted, Xelon assigns one automatically.",
							Optional: true,
							Computed: true,
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
				MarkdownDescription: "The password for the device root or administrator user. Required if `user_data` is empty.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
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
			"swap_disk_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the swap disk.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"swap_disk_size": schema.Int64Attribute{
				MarkdownDescription: "The size of the swap disk in GB. Required if `user_data` is empty.",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					helper.ExpandOnlyStorageSizeModifier(),
				},
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
			"user_data": schema.StringAttribute{
				MarkdownDescription: "User data to provide when launching the device. Updates to this field will force a new resource to be created.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
	if data.UserData.ValueString() != "" {
		createRequest.CloudInit = &xelon.DeviceCloudInit{
			UserData: data.UserData.ValueString(),
		}
	}
	if data.CPUCoreHotPlug.ValueBool() {
		createRequest.EnableCPUCoresHotAdd = data.CPUCoreHotPlug.ValueBool()
	}
	if data.MemoryHotPlug.ValueBool() {
		createRequest.EnableRAMHotAdd = data.MemoryHotPlug.ValueBool()
	}
	tflog.Debug(ctx, "creating device", map[string]any{
		"display_name": data.DisplayName.ValueString(),
		"hostname":     data.Hostname.ValueString(),
		"template_id":  data.TemplateID.ValueString(),
		"tenant_id":    data.TenantID.ValueString(),
	})
	createdDevice, _, err := r.client.Devices.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create device", err.Error())
		return
	}
	tflog.Debug(ctx, "created device", map[string]any{"data": createdDevice})

	deviceID := createdDevice.ID

	tflog.Info(ctx, "waiting for device to be powered on", map[string]any{"device_id": deviceID})
	err = helper.WaitDevicePowerStateOn(ctx, r.client, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for device to be powered on", err.Error())
		return
	}
	tflog.Info(ctx, "device is powered on", map[string]any{"device_id": deviceID})

	tflog.Info(ctx, "waiting for device to be ready", map[string]any{"device_id": deviceID})
	err = helper.WaitDeviceStateReady(ctx, r.client, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for device to be ready", err.Error())
		return
	}
	tflog.Info(ctx, "device is ready", map[string]any{"device_id": deviceID})

	_, err = r.refreshDeviceState(ctx, &data, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to refresh device state", err.Error())
		return
	}

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
	resp, err := r.refreshDeviceState(ctx, &data, deviceID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the tag is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to refresh device state", err.Error())
		return
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *deviceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state deviceResourceModel

	// read plan and state data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	deviceID := state.ID.ValueString()

	if !plan.DisplayName.Equal(state.DisplayName) {
		updateRequest := &xelon.DeviceUpdateRequest{
			DisplayName: plan.DisplayName.ValueString(),
		}
		tflog.Debug(ctx, "updating device name", map[string]any{"device_id": deviceID, "payload": updateRequest})
		updatedDevice, _, err := r.client.Devices.Update(ctx, deviceID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update device name", err.Error())
			return
		}
		tflog.Debug(ctx, "updated device name", map[string]any{"data": updatedDevice})

		plan.DisplayName = types.StringValue(updatedDevice.DisplayName)
	}

	if !plan.DiskSize.Equal(state.DiskSize) {
		err := deleteSnapshotsIfNeeded(ctx, r.client, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to delete device disk snapshots", err.Error())
			return
		}

		diskID := plan.DiskID.ValueString()
		newDiskSize := int(plan.DiskSize.ValueInt64())
		updateRequest := &xelon.DeviceUpdateDiskRequest{
			CreateSnapshot:  true,
			DiskID:          diskID,
			ExtendPartition: true,
			Size:            newDiskSize,
		}
		tflog.Debug(ctx, "updating disk size", map[string]any{"device_id": deviceID, "payload": updateRequest})
		device, _, err := r.client.Devices.UpdateDisk(ctx, deviceID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update disk size", err.Error())
			return
		}
		tflog.Debug(ctx, "updated disk size", map[string]any{"device_id": deviceID, "data": device})

		tflog.Info(ctx, "waiting for disk size to be updated after extension")
		err = helper.WaitDeviceStateReady(ctx, r.client, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to wait for disk size to be updated", err.Error())
			return
		}
		tflog.Info(ctx, "disk is updated after extension")
	}

	if !plan.SwapDiskSize.Equal(state.SwapDiskSize) {
		err := deleteSnapshotsIfNeeded(ctx, r.client, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to delete device disk snapshots", err.Error())
			return
		}

		swapDiskID := plan.SwapDiskID.ValueString()
		newSwapDiskSize := int(plan.SwapDiskSize.ValueInt64())
		updateRequest := &xelon.DeviceUpdateDiskRequest{
			CreateSnapshot:  true,
			DiskID:          swapDiskID,
			ExtendPartition: true,
			Size:            newSwapDiskSize,
		}
		tflog.Debug(ctx, "updating swap disk size", map[string]any{"device_id": deviceID, "payload": updateRequest})
		device, _, err := r.client.Devices.UpdateDisk(ctx, deviceID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update swap disk size", err.Error())
			return
		}
		tflog.Debug(ctx, "updated swap disk size", map[string]any{"device_id": deviceID, "data": device})

		tflog.Info(ctx, "waiting for swap disk size to be updated after extension")
		err = helper.WaitDeviceStateReady(ctx, r.client, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to wait for swap disk size to be updated", err.Error())
			return
		}
		tflog.Info(ctx, "swap disk is updated after extension")
	}

	if !plan.CPUCoreCount.Equal(state.CPUCoreCount) || !plan.Memory.Equal(state.Memory) {
		// device must be stopped before changing CPU count and RAM if hotplug is false
		deviceMustBeRestarted := !state.CPUCoreHotPlug.ValueBool() || !state.MemoryHotPlug.ValueBool()
		if deviceMustBeRestarted {
			tflog.Debug(ctx, "getting device", map[string]any{"device_id": deviceID})
			device, _, err := r.client.Devices.Get(ctx, deviceID)
			if err != nil {
				response.Diagnostics.AddError("Unable to get device", err.Error())
				return
			}
			tflog.Debug(ctx, "got device", map[string]any{"data": device})
			if device.PoweredOn {
				tflog.Debug(ctx, "stopping device", map[string]any{"device_id": deviceID})
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
		}

		updateRequest := &xelon.DeviceUpdateHardwareRequest{
			CPUCores: int(plan.CPUCoreCount.ValueInt64()),
			RAM:      int(plan.Memory.ValueInt64()),
		}
		tflog.Debug(ctx, "updating device hardware", map[string]any{"device_id": deviceID, "payload": updateRequest})
		updatedDevice, _, err := r.client.Devices.UpdateHardware(ctx, deviceID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update device hardware", err.Error())
			return
		}
		tflog.Debug(ctx, "updated device hardware", map[string]any{"data": updatedDevice})

		// device must be started after changing CUP count and RAM if hotplug is false
		if deviceMustBeRestarted {
			tflog.Debug(ctx, "getting device", map[string]any{"device_id": deviceID})
			device, _, err := r.client.Devices.Get(ctx, deviceID)
			if err != nil {
				response.Diagnostics.AddError("Unable to get device", err.Error())
				return
			}
			tflog.Debug(ctx, "got device", map[string]any{"data": device})
			if !device.PoweredOn {
				tflog.Debug(ctx, "starting device", map[string]any{"device_id": deviceID})
				_, err := r.client.Devices.Start(ctx, deviceID)
				if err != nil {
					response.Diagnostics.AddError("Unable to start device", err.Error())
					return
				}

				err = helper.WaitDevicePowerStateOn(ctx, r.client, deviceID)
				if err != nil {
					response.Diagnostics.AddError("Unable to wait for device to be powered on", err.Error())
					return
				}
			}
		}

		// ensure device is in ready state
		tflog.Info(ctx, "waiting for device to be ready", map[string]any{"device_id": deviceID})
		err = helper.WaitDeviceStateReady(ctx, r.client, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to wait for device to be ready", err.Error())
			return
		}
		tflog.Info(ctx, "device is ready", map[string]any{"device_id": deviceID})
	}

	_, err := r.refreshDeviceState(ctx, &plan, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to refresh device state", err.Error())
		return
	}

	diags := response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
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
	tflog.Debug(ctx, "getting device", map[string]any{"device_id": deviceID})
	device, resp, err := r.client.Devices.Get(ctx, deviceID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return
		}
		response.Diagnostics.AddError("Unable to get device", err.Error())
		return
	}
	tflog.Debug(ctx, "got device", map[string]any{"data": device})

	if device.PoweredOn {
		tflog.Info(ctx, "stopping device", map[string]any{"data": device})
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

	tflog.Debug(ctx, "deleting device", map[string]any{"device_id": deviceID})
	_, err = r.client.Devices.Delete(ctx, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete device", err.Error())
		return
	}
	tflog.Debug(ctx, "deleted device", map[string]any{"device_id": deviceID})
}

func (r *deviceResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func (r *deviceResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	if request.Plan.Raw.IsNull() {
		return
	}

	var plan deviceResourceModel
	diags := request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	requiresCreateInputs := request.State.Raw.IsNull()
	if !request.State.Raw.IsNull() {
		var state deviceResourceModel
		response.Diagnostics.Append(request.State.Get(ctx, &state)...)
		if response.Diagnostics.HasError() {
			return
		}

		requiresCreateInputs = !plan.Password.Equal(state.Password) ||
			!plan.TemplateID.Equal(state.TemplateID) ||
			!plan.UserData.Equal(state.UserData)
	}
	if !requiresCreateInputs {
		return
	}

	if plan.UserData.IsUnknown() || plan.UserData.ValueString() != "" {
		return
	}
	if plan.Password.IsUnknown() || plan.SwapDiskSize.IsUnknown() {
		return
	}
	if plan.Password.ValueString() == "" {
		response.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing password or user_data",
			`Either "password" or "user_data" must be specified when creating or replacing a device.`,
		)
	}
	if plan.SwapDiskSize.IsNull() {
		response.Diagnostics.AddAttributeError(
			path.Root("swap_disk_size"),
			"Missing required attribute",
			`Attribute "swap_disk_size" must be specified when "user_data" is not set.`,
		)
	}
}

func (m *deviceResourceModel) fromAPI(ctx context.Context, device *xelon.Device, deviceNetworks []xelon.DeviceNetwork) {
	primaryDisk := findDiskIDBySize(ctx, int(m.DiskSize.ValueInt64()), device.Storages)
	swapDisk := findDiskIDBySize(ctx, int(m.SwapDiskSize.ValueInt64()), device.Storages)

	if primaryDisk != nil {
		m.DiskID = types.StringValue(primaryDisk.ID)
	}
	if swapDisk != nil {
		m.SwapDiskID = types.StringValue(swapDisk.ID)
	}
	m.CPUCoreCount = types.Int64Value(int64(device.CPUCores))
	m.CPUCoreHotPlug = types.BoolValue(device.CPUCoresHotAddEnabled)
	m.DisplayName = types.StringValue(device.DisplayName)
	m.EnableMonitoring = types.BoolValue(false)
	m.Hostname = types.StringValue(device.HostName)
	m.ID = types.StringValue(device.ID)
	m.Memory = types.Int64Value(int64(device.RAM))
	m.MemoryHotPlug = types.BoolValue(device.RAMHotAddEnabled)
	m.Networks = populateDeviceNetworkIPv4Addresses(m.Networks, deviceNetworks)
}

func (r *deviceResource) refreshDeviceState(ctx context.Context, model *deviceResourceModel, deviceID string) (*xelon.Response, error) {
	tflog.Debug(ctx, "refreshing device state", map[string]any{"device_id": deviceID})

	tflog.Trace(ctx, "getting device via API (state refresh)", map[string]any{"device_id": deviceID})
	device, resp, err := r.client.Devices.Get(ctx, deviceID)
	if err != nil {
		return resp, err
	}
	tflog.Trace(ctx, "received device from API", map[string]any{
		"device_id": deviceID,
		"data":      device,
	})

	tflog.Trace(ctx, "getting device network info via API (state refresh)", map[string]any{"device_id": deviceID})
	deviceNetworks, _, err := r.client.Devices.GetNetworkInfo(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	tflog.Trace(ctx, "received device network info from API", map[string]any{
		"device_id":     deviceID,
		"network_count": len(deviceNetworks),
	})

	model.fromAPI(ctx, device, deviceNetworks)

	return nil, nil
}

func populateDeviceNetworkIPv4Addresses(networks []deviceNetworkResourceModel, deviceNetworks []xelon.DeviceNetwork) []deviceNetworkResourceModel {
	for i := range networks {
		networks[i].IPAddress = types.StringNull()
		networkID := networks[i].ID.ValueString()

		for _, deviceNetwork := range deviceNetworks {
			if deviceNetwork.ID != networkID || !deviceNetwork.Connected {
				continue
			}

			for _, ipAddress := range deviceNetwork.IPAddresses {
				if ipAddress.Is4() {
					networks[i].IPAddress = types.StringValue(ipAddress.String())
					break
				}
			}
			if !networks[i].IPAddress.IsNull() {
				break
			}
		}
	}

	return networks
}

// findDiskIDBySize looks up for xelon.DeviceStorage by size. If multiple disks are found,
// the disk with lower unit_number is preferred.
func findDiskIDBySize(ctx context.Context, diskSize int, storages []xelon.DeviceStorage) *xelon.DeviceStorage {
	var storagesMatchedBySize []*xelon.DeviceStorage
	for _, storage := range storages {
		if storage.Size == diskSize {
			storagesMatchedBySize = append(storagesMatchedBySize, &storage)
		}
	}
	if len(storagesMatchedBySize) == 0 {
		tflog.Warn(ctx, "no disk with requested size was found", map[string]any{"disk_size": diskSize, "storages": storages})
		return nil
	}

	slices.SortFunc(storagesMatchedBySize, func(first *xelon.DeviceStorage, second *xelon.DeviceStorage) int {
		return first.UnitNumber - second.UnitNumber
	})
	return storagesMatchedBySize[0]
}

func deleteSnapshotsIfNeeded(ctx context.Context, client *xelon.Client, deviceID string) error {
	snapshots, _, err := client.Snapshots.List(ctx, deviceID, nil)
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		return nil
	}

	tflog.Info(ctx, "deleting snapshots for device", map[string]any{
		"device_id": deviceID,
		"snapshots": snapshots,
	})
	var errs []error
	for _, snapshot := range snapshots {
		_, err := client.Snapshots.Delete(ctx, deviceID, snapshot.ID, &xelon.SnapshotDeleteRequest{RemoveChildSnapshots: true})
		if err != nil {
			tflog.Error(ctx, "unable to delete snapshot", map[string]any{
				"device_id":   deviceID,
				"snapshot_id": snapshot.ID,
			})
			errs = append(errs, err)
		}
		tflog.Debug(ctx, "deleted snapshot", map[string]any{
			"device_id":   deviceID,
			"snapshot_id": snapshot.ID,
		})
	}

	err = helper.WaitDeviceSnapshotsDeleted(ctx, client, deviceID)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
