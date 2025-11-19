package provider

import (
	"context"
	"fmt"

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

	// Update disk sizes if changed
	if !plan.DiskSize.Equal(state.DiskSize) || !plan.SwapDiskSize.Equal(state.SwapDiskSize) {
		// Get current device state to access disk IDs
		tflog.Debug(ctx, "Getting device state for disk update", map[string]any{"device_id": deviceID})
		device, _, err := r.client.Devices.Get(ctx, deviceID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get device", err.Error())
			return
		}
		tflog.Debug(ctx, "Got device for disk update", map[string]any{"data": device})

		// Build a map of unitNumber -> storage for easy lookup
		storageByUnit := make(map[int]*xelon.DeviceStorage)
		for i := range device.Storages {
			storageByUnit[device.Storages[i].UnitNumber] = &device.Storages[i]
		}

		// Update main disk (unitNumber 0) if size changed
		if !plan.DiskSize.Equal(state.DiskSize) {
			if mainDisk, ok := storageByUnit[0]; ok {
				newSize := int(plan.DiskSize.ValueInt64())

				// Validate that we're not attempting to downsize the disk
				if newSize < mainDisk.Size {
					response.Diagnostics.AddError(
						"Cannot downsize disk",
						fmt.Sprintf("The main disk (disk 0) cannot be made smaller than its current size. Current size: %d GB, requested size: %d GB. Disk shrinking is not supported.", mainDisk.Size, newSize),
					)
					return
				}

				tflog.Info(ctx, "Updating main disk size", map[string]any{
					"device_id": deviceID,
					"disk_id":   mainDisk.ID,
					"old_size":  mainDisk.Size,
					"new_size":  newSize,
				})

				diskUpdateRequest := &xelon.DeviceUpdateDiskRequest{
					DiskID:          mainDisk.ID,
					Size:            newSize,
					ExtendPartition: false,
					CreateSnapshot:  false,
				}
				_, err := r.client.Devices.UpdateDisk(ctx, deviceID, diskUpdateRequest)
				if err != nil {
					response.Diagnostics.AddError("Unable to update main disk size", err.Error())
					return
				}
				tflog.Info(ctx, "Updated main disk size successfully")
			}
		}

		// Update swap disk (unitNumber 1) if size changed
		if !plan.SwapDiskSize.Equal(state.SwapDiskSize) {
			if swapDisk, ok := storageByUnit[1]; ok {
				newSize := int(plan.SwapDiskSize.ValueInt64())

				// Validate that we're not attempting to downsize the disk
				if newSize < swapDisk.Size {
					response.Diagnostics.AddError(
						"Cannot downsize disk",
						fmt.Sprintf("The swap disk (disk 1) cannot be made smaller than its current size. Current size: %d GB, requested size: %d GB. Disk shrinking is not supported.", swapDisk.Size, newSize),
					)
					return
				}

				tflog.Info(ctx, "Updating swap disk size", map[string]any{
					"device_id": deviceID,
					"disk_id":   swapDisk.ID,
					"old_size":  swapDisk.Size,
					"new_size":  newSize,
				})

				diskUpdateRequest := &xelon.DeviceUpdateDiskRequest{
					DiskID:          swapDisk.ID,
					Size:            newSize,
					ExtendPartition: false,
					CreateSnapshot:  false,
				}
				_, err := r.client.Devices.UpdateDisk(ctx, deviceID, diskUpdateRequest)
				if err != nil {
					response.Diagnostics.AddError("Unable to update swap disk size", err.Error())
					return
				}
				tflog.Info(ctx, "Updated swap disk size successfully")
			}
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

	// Map storage devices to disk_size and swap_disk_size
	// Typically: first storage (unitNumber 0) is main disk, second (unitNumber 1) is swap
	for _, storage := range device.Storages {
		if storage.UnitNumber == 0 {
			plan.DiskSize = types.Int64Value(int64(storage.Size))
		} else if storage.UnitNumber == 1 {
			plan.SwapDiskSize = types.Int64Value(int64(storage.Size))
		}
	}

	if device.Template != nil {
		plan.TemplateID = types.StringValue(device.Template.ID)
	}
	if device.Tenant != nil {
		plan.TenantID = types.StringValue(device.Tenant.ID)
	}

	// Preserve fields from state that aren't being updated
	plan.Networks = state.Networks
	plan.BackupJobID = state.BackupJobID
	plan.SSHKeyID = state.SSHKeyID
	plan.ScriptID = state.ScriptID
	plan.SendEmail = state.SendEmail
	plan.Password = state.Password

	diags := response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}
