package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func (r *deviceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state deviceResourceModel

	// Read plan and state data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	deviceID := state.ID.ValueString()
	needsRestart := false
	var wasPoweredOn bool

	// Update display name if changed
	if !plan.DisplayName.Equal(state.DisplayName) {
		updateRequest := &xelon.DeviceUpdateRequest{
			DisplayName: plan.DisplayName.ValueString(),
		}
		tflog.Debug(ctx, "Updating device name", map[string]any{"device_id": deviceID, "payload": updateRequest})
		updatedDevice, _, err := r.client.Devices.Update(ctx, deviceID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update device name", err.Error())
			return
		}
		tflog.Debug(ctx, "Updated device name", map[string]any{"data": updatedDevice})
	}

	// Update CPU/RAM if changed (requires device to be powered off)
	if !plan.CPUCoreCount.Equal(state.CPUCoreCount) || !plan.Memory.Equal(state.Memory) {
		// Get current device state
		tflog.Debug(ctx, "Getting device state for hardware update", map[string]any{"device_id": deviceID})
		device, _, err := r.client.Devices.Get(ctx, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get device", err.Error())
			return
		}
		tflog.Debug(ctx, "Got device", map[string]any{"data": device})

		wasPoweredOn = device.PoweredOn

		// Device must be stopped before changing CPU count and RAM
		if device.PoweredOn {
			tflog.Info(ctx, "Stopping device for hardware update", map[string]any{"device_id": deviceID})
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
			needsRestart = true
		}

		// Update hardware
		updateRequest := &xelon.DeviceUpdateHardwareRequest{
			CPUCores: int(plan.CPUCoreCount.ValueInt64()),
			RAM:      int(plan.Memory.ValueInt64()),
		}
		tflog.Debug(ctx, "Updating device hardware", map[string]any{"device_id": deviceID, "payload": updateRequest})
		_, _, err = r.client.Devices.UpdateHardware(ctx, deviceID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update device hardware", err.Error())
			return
		}
		tflog.Debug(ctx, "Updated device hardware successfully")
	}

	// Restart device if it was powered on before hardware update
	if needsRestart && wasPoweredOn {
		tflog.Info(ctx, "Starting device after hardware update", map[string]any{"device_id": deviceID})
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

	// Get updated device state to refresh computed attributes
	tflog.Debug(ctx, "Getting device with updated data", map[string]any{"device_id": deviceID})
	device, _, err := r.client.Devices.Get(ctx, deviceID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get device", err.Error())
		return
	}
	tflog.Debug(ctx, "Got updated device", map[string]any{"data": device})

	// Update state with actual values from API
	plan.DisplayName = types.StringValue(device.DisplayName)
	plan.CPUCoreCount = types.Int64Value(int64(device.CPUCores))
	plan.Memory = types.Int64Value(int64(device.RAM))
	plan.EnableMonitoring = types.BoolValue(device.MonitoringEnabled)
	if device.Template != nil {
		plan.TemplateID = types.StringValue(device.Template.ID)
	}
	if device.Tenant != nil {
		plan.TenantID = types.StringValue(device.Tenant.ID)
	}

	// Preserve fields from state that aren't being updated
	plan.Networks = state.Networks
	plan.DiskSize = state.DiskSize
	plan.SwapDiskSize = state.SwapDiskSize
	plan.BackupJobID = state.BackupJobID
	plan.SSHKeyID = state.SSHKeyID
	plan.ScriptID = state.ScriptID
	plan.SendEmail = state.SendEmail
	plan.Password = state.Password

	diags := response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}
