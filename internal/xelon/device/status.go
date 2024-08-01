package device

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	devicePowerStateOn  = "PoweredOn"
	devicePowerStateOff = "PoweredOff"

	deviceVMWareToolsStatusRunning    = "Running"
	deviceVMWareToolsStatusNotRunning = "NotRunning"
)

func statusPowerState(ctx context.Context, client *xelon.Client, tenantID, localVMID string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		deviceRoot, resp, err := client.Devices.Get(ctx, tenantID, localVMID)
		if err != nil {
			// API returns 503 for devices in-provisioning state
			if resp != nil && resp.StatusCode == 503 {
				return deviceRoot, devicePowerStateOff, nil
			}
			return nil, "", err
		}

		if deviceRoot.Device == nil {
			return nil, "", fmt.Errorf("parsing Device information (localVMID: %s)", localVMID)
		}
		if deviceRoot.Device.LocalVMDetails == nil {
			return nil, "", fmt.Errorf("parsing Device.LocalVMDetails information (localVMID: %s)", localVMID)
		}

		deviceState := devicePowerStateOff
		if deviceRoot.Device.PowerState && deviceRoot.Device.LocalVMDetails.State == 1 {
			deviceState = devicePowerStateOn
		} else if !deviceRoot.Device.PowerState {
			deviceState = devicePowerStateOff
		}

		return deviceRoot, deviceState, nil
	}
}

func statusVMWareToolsStatus(ctx context.Context, client *xelon.Client, tenantID, localVMID string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		deviceRoot, _, err := client.Devices.Get(ctx, tenantID, localVMID)
		if err != nil {
			return nil, "", err
		}

		if deviceRoot.ToolsStatus == nil {
			return nil, "", fmt.Errorf("parsing ToolsStatus information (localVMID: %s)", localVMID)
		}

		deviceToolsStatus := deviceVMWareToolsStatusNotRunning
		if deviceRoot.ToolsStatus.ToolsStatus && deviceRoot.ToolsStatus.RunningStatus == "guestToolsRunning" {
			deviceToolsStatus = deviceVMWareToolsStatusRunning
		} else if !deviceRoot.ToolsStatus.ToolsStatus {
			tflog.Debug(ctx, "VMWare Tools are not installed, returning status running.")
			deviceToolsStatus = deviceVMWareToolsStatusRunning
		}

		return deviceRoot, deviceToolsStatus, nil
	}
}
