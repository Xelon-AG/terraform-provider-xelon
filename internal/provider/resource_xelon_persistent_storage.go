package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*persistentStorageResource)(nil)
	_ resource.ResourceWithConfigure   = (*persistentStorageResource)(nil)
	_ resource.ResourceWithImportState = (*persistentStorageResource)(nil)
)

// persistentStorageResource is the persistent storage resource implementation.
type persistentStorageResource struct {
	client *xelon.Client
}

// persistentStorageResourceModel maps the persistent storage resource schema data.
type persistentStorageResourceModel struct {
	CloudID  types.String `tfsdk:"cloud_id"`
	DeviceID types.String `tfsdk:"device_id"`
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Size     types.Int64  `tfsdk:"size"`
	TenantID types.String `tfsdk:"tenant_id"`
	UUID     types.String `tfsdk:"uuid"`
}

func NewPersistentStorageResource() resource.Resource {
	return &persistentStorageResource{}
}

func (r *persistentStorageResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_persistent_storage"
}

func (r *persistentStorageResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The persistent storage resource allows you to manage Xelon persistent storages.

Persistent storages are detachable storages for your Devices and clusters.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the device to which the persistent storage will be connected.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the persistent storage.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The persistent storage name.",
				Required:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "The size of the persistent storage in GB.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					helper.ExpandOnlyStorageSizeModifier(),
				},
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the persistent storage belongs.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the persistent storage device.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *persistentStorageResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *persistentStorageResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data persistentStorageResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &xelon.PersistentStorageCreateRequest{
		Name: data.Name.ValueString(),
		Size: int(data.Size.ValueInt64()),
		Type: 2,
	}
	if data.CloudID.ValueString() != "" {
		createRequest.CloudID = data.CloudID.ValueString()
	}
	if data.DeviceID.ValueString() != "" {
		createRequest.DeviceID = data.DeviceID.ValueString()
	}
	if data.TenantID.ValueString() != "" {
		createRequest.TenantID = data.TenantID.ValueString()
	}
	tflog.Debug(ctx, "Creating persistent storage", map[string]any{"payload": createRequest})
	createdPersistentStorage, _, err := r.client.PersistentStorages.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create persistent storage", err.Error())
		return
	}
	tflog.Debug(ctx, "Created persistent storage", map[string]any{"data": createdPersistentStorage})

	persistentStorageID := createdPersistentStorage.ID

	tflog.Info(ctx, "Waiting for persistent storage to be formatted", map[string]any{"persistent_storage_id": persistentStorageID})
	err = helper.WaitPersistentStorageStateFormatted(ctx, r.client, persistentStorageID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for persistent storage to be formatted", err.Error())
		return
	}
	tflog.Info(ctx, "Persistent storage is formatted", map[string]any{"persistent_storage_id": persistentStorageID})

	tflog.Info(ctx, "Waiting for persistent storage to be ready", map[string]any{"persistent_storage_id": persistentStorageID})
	err = helper.WaitPersistentStorageStateReady(ctx, r.client, persistentStorageID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for persistent storage to be ready", err.Error())
		return
	}
	tflog.Info(ctx, "Persistent storage is ready", map[string]any{"persistent_storage_id": persistentStorageID})

	tflog.Debug(ctx, "Getting persistent storage with enriched data", map[string]any{"persistent_storage_id": persistentStorageID})
	persistentStorage, _, err := r.client.PersistentStorages.Get(ctx, persistentStorageID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get persistent storage", err.Error())
		return
	}
	tflog.Debug(ctx, "Got persistent storage with enriched data", map[string]any{"data": persistentStorage})

	// map response body to attributes
	if persistentStorage.Cloud != nil {
		data.CloudID = types.StringValue(persistentStorage.Cloud.ID)
	}
	if len(persistentStorage.AttachedDevices) > 0 {
		// at the moment API allows to attach only one device
		data.DeviceID = types.StringValue(persistentStorage.AttachedDevices[0].ID)
	} else {
		data.DeviceID = types.StringValue("")
	}
	data.ID = types.StringValue(persistentStorage.ID)
	data.Name = types.StringValue(persistentStorage.Name)
	data.Size = types.Int64Value(int64(persistentStorage.Capacity))
	if persistentStorage.Tenant != nil {
		data.TenantID = types.StringValue(persistentStorage.Tenant.ID)
	}
	data.UUID = types.StringValue(persistentStorage.UUID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *persistentStorageResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data persistentStorageResourceModel

	// read plan data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	persistentStorageID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting persistent storage", map[string]any{"persistent_storage_id": persistentStorageID})
	persistentStorage, resp, err := r.client.PersistentStorages.Get(ctx, persistentStorageID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the persistent storage is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get persistent storage", err.Error())
		return
	}
	tflog.Debug(ctx, "Got persistent storage", map[string]any{"data": persistentStorage})

	// map response body to attributes
	if persistentStorage.Cloud != nil {
		data.CloudID = types.StringValue(persistentStorage.Cloud.ID)
	}
	if len(persistentStorage.AttachedDevices) > 0 {
		// at the moment API allows to attach only one device
		data.DeviceID = types.StringValue(persistentStorage.AttachedDevices[0].ID)
	} else {
		data.DeviceID = types.StringValue("")
	}
	data.ID = types.StringValue(persistentStorage.ID)
	data.Name = types.StringValue(persistentStorage.Name)
	data.Size = types.Int64Value(int64(persistentStorage.Capacity))
	if persistentStorage.Tenant != nil {
		data.TenantID = types.StringValue(persistentStorage.Tenant.ID)
	}
	data.UUID = types.StringValue(persistentStorage.UUID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *persistentStorageResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state persistentStorageResourceModel

	// read plan and state data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	persistentStorageID := state.ID.ValueString()

	if !plan.DeviceID.Equal(state.DeviceID) {
		newDeviceID := plan.DeviceID.ValueString()
		oldDeviceID := state.DeviceID.ValueString()

		// detach if needed
		if oldDeviceID != "" {
			tflog.Debug(ctx, "Detaching persistent storage from device", map[string]any{"persistent_storage_id": persistentStorageID, "device_id": oldDeviceID})
			_, err := r.client.PersistentStorages.DetachFromDevice(ctx, persistentStorageID, oldDeviceID)
			if err != nil {
				response.Diagnostics.AddError("Unable to detach persistent storage", err.Error())
				return
			}
			tflog.Debug(ctx, "Detached persistent storage from device", map[string]any{"persistent_storage_id": persistentStorageID, "device_id": oldDeviceID})
			plan.DeviceID = types.StringValue("")
		}
		// attach if needed
		if newDeviceID != "" {
			tflog.Debug(ctx, "Attaching persistent storage to device", map[string]any{"persistent_storage_id": persistentStorageID, "device_id": oldDeviceID})
			_, err := r.client.PersistentStorages.AttachToDevice(ctx, persistentStorageID, newDeviceID)
			if err != nil {
				response.Diagnostics.AddError("Unable to attach persistent storage", err.Error())
				return
			}
			tflog.Debug(ctx, "Attached persistent storage to device", map[string]any{"persistent_storage_id": persistentStorageID, "device_id": oldDeviceID})
			plan.DeviceID = types.StringValue(newDeviceID)
		}
	}

	if !plan.Size.Equal(state.Size) {
		newStorageSize := int(plan.Size.ValueInt64())
		tflog.Debug(ctx, "Extending persistent storage size", map[string]any{"persistent_storage_id": persistentStorageID})
		_, err := r.client.PersistentStorages.Extend(ctx, persistentStorageID, newStorageSize)
		if err != nil {
			response.Diagnostics.AddError("Unable to extend persistent storage size", err.Error())
			return
		}
		tflog.Debug(ctx, "Extended persistent storage size", map[string]any{"persistent_storage_id": persistentStorageID})

		// TODO: API doesn't return new size, waiting for fix

		tflog.Debug(ctx, "Getting persistent storage with enriched data", map[string]any{"persistent_storage_id": persistentStorageID})
		persistentStorage, _, err := r.client.PersistentStorages.Get(ctx, persistentStorageID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get persistent storage", err.Error())
			return
		}
		tflog.Debug(ctx, "Got persistent storage with enriched data", map[string]any{"data": persistentStorage})

		plan.Size = types.Int64Value(int64(persistentStorage.Capacity))
	}

	diags := response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}

func (r *persistentStorageResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data persistentStorageResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	persistentStorageID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting persistent storage", map[string]any{"persistent_storage_id": persistentStorageID})
	_, err := r.client.PersistentStorages.Delete(ctx, persistentStorageID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete persistent storage", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted persistent storage", map[string]any{"persistent_storage_id": persistentStorageID})
}

func (r *persistentStorageResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
