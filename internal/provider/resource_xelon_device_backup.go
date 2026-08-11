package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*deviceBackupResource)(nil)
	_ resource.ResourceWithConfigure   = (*deviceBackupResource)(nil)
	_ resource.ResourceWithImportState = (*deviceBackupResource)(nil)
)

// deviceBackupResource is the device backup resource implementation.
type deviceBackupResource struct {
	client *xelon.Client
}

// deviceBackupResourceModel maps the device backup resource schema data.
type deviceBackupResourceModel struct {
	BackupPlanID types.Int64  `tfsdk:"backup_plan_id"`
	DeviceID     types.String `tfsdk:"device_id"`
}

func NewDeviceBackupResource() resource.Resource {
	return &deviceBackupResource{}
}

func (r *deviceBackupResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_device_backup"
}

func (r *deviceBackupResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The device backup resource manages the assignment of one cloud-wide backup plan to a Xelon device.

Deleting this resource disables backups for the device by removing its backup plan assignment. It does not delete the device or the backup plan.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"backup_plan_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the backup plan assigned to the device.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"device_id": schema.StringAttribute{
				MarkdownDescription: "ID of the device whose backup plan assignment is managed. Changing this value requires replacing the resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *deviceBackupResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *deviceBackupResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data deviceBackupResourceModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	deviceID := data.DeviceID.ValueString()
	backupPlanID := int(data.BackupPlanID.ValueInt64())

	tflog.Debug(ctx, "creating device backup assignment", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlanID,
	})

	tflog.Trace(ctx, "assigning device backup plan via API", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlanID,
	})
	_, err := r.client.Backups.SetDevicePlan(ctx, deviceID, backupPlanID)
	if err != nil {
		response.Diagnostics.AddError("Unable to create device backup assignment", err.Error())
		return
	}

	tflog.Debug(ctx, "assigned device backup plan", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlanID,
	})

	tflog.Debug(ctx, "refreshing device backup assignment state after create", map[string]any{"device_id": deviceID})

	tflog.Trace(ctx, "reading device backup plan via API (created assignment refresh)", map[string]any{"device_id": deviceID})
	backupPlan, _, err := r.client.Backups.GetDevicePlan(ctx, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to read device backup assignment", err.Error())
		return
	}
	if backupPlan == nil {
		response.Diagnostics.AddError(
			"Unable to read device backup assignment",
			"The backup plan was assigned, but no assignment was found while refreshing state.",
		)
		return
	}
	tflog.Trace(ctx, "received device backup plan from API", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlan.ID,
	})

	data.fromAPI(backupPlan, deviceID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *deviceBackupResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data deviceBackupResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	deviceID := data.DeviceID.ValueString()

	tflog.Debug(ctx, "reading device backup assignment", map[string]any{"device_id": deviceID})

	tflog.Trace(ctx, "reading device backup plan via API", map[string]any{"device_id": deviceID})
	backupPlan, resp, err := r.client.Backups.GetDevicePlan(ctx, deviceID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read device backup assignment", err.Error())
		return
	}
	if backupPlan == nil {
		response.State.RemoveResource(ctx)
		return
	}
	tflog.Trace(ctx, "received device backup plan from API", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlan.ID,
	})

	data.fromAPI(backupPlan, deviceID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *deviceBackupResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data deviceBackupResourceModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	deviceID := data.DeviceID.ValueString()
	backupPlanID := int(data.BackupPlanID.ValueInt64())

	tflog.Debug(ctx, "updating device backup assignment", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlanID,
	})

	tflog.Trace(ctx, "assigning device backup plan via API", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlanID,
	})
	_, err := r.client.Backups.SetDevicePlan(ctx, deviceID, backupPlanID)
	if err != nil {
		response.Diagnostics.AddError("Unable to update device backup assignment", err.Error())
		return
	}

	tflog.Debug(ctx, "assigned device backup plan", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlanID,
	})

	tflog.Debug(ctx, "refreshing device backup assignment state after update", map[string]any{"device_id": deviceID})

	tflog.Trace(ctx, "reading device backup plan via API (updated assignment refresh)", map[string]any{"device_id": deviceID})
	backupPlan, _, err := r.client.Backups.GetDevicePlan(ctx, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to read device backup assignment", err.Error())
		return
	}
	if backupPlan == nil {
		response.Diagnostics.AddError(
			"Unable to read device backup assignment",
			"The backup plan was updated, but no assignment was found while refreshing state.",
		)
		return
	}
	tflog.Trace(ctx, "received device backup plan from API", map[string]any{
		"device_id":      deviceID,
		"backup_plan_id": backupPlan.ID,
	})

	data.fromAPI(backupPlan, deviceID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *deviceBackupResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data deviceBackupResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	deviceID := data.DeviceID.ValueString()

	tflog.Debug(ctx, "deleting device backup assignment", map[string]any{"device_id": deviceID})

	tflog.Trace(ctx, "disabling device backup via API", map[string]any{"device_id": deviceID})
	resp, err := r.client.Backups.DisableDeviceBackup(ctx, deviceID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "device backup assignment already absent", map[string]any{"device_id": deviceID})
			return
		}
		response.Diagnostics.AddError("Unable to delete device backup assignment", err.Error())
		return
	}

	tflog.Debug(ctx, "disabled device backup", map[string]any{"device_id": deviceID})
}

func (r *deviceBackupResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("device_id"), request.ID)...)
}

func (m *deviceBackupResourceModel) fromAPI(backupPlan *xelon.BackupPlan, deviceID string) {
	m.BackupPlanID = types.Int64Value(int64(backupPlan.ID))
	m.DeviceID = types.StringValue(deviceID)
}
