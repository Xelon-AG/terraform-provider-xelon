package helper

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	deviceStatePowerOn  = "poweredOn"
	deviceStatePowerOff = "poweredOff"

	deviceStateProvisioning     = "provisioning"
	deviceStateReadyForBasicUse = "readyForBasicUse"
	deviceStateReady            = "ready"
)

func statusDevicePowerState(ctx context.Context, client *xelon.Client, deviceID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		device, resp, err := client.Devices.Get(ctx, deviceID)
		if err != nil {
			// API returns 500 sometimes for fresh created devices in-provisioning state
			if resp != nil && resp.StatusCode == http.StatusInternalServerError {
				return device, deviceStatePowerOff, nil
			}
			return nil, "", err
		}
		if device == nil {
			return nil, "", fmt.Errorf("failed to get device with id: %s", deviceID)
		}

		var devicePowerState string
		if device.PoweredOn {
			devicePowerState = deviceStatePowerOn
		} else {
			devicePowerState = deviceStatePowerOff
		}

		return device, devicePowerState, nil
	}
}

func statusDeviceState(ctx context.Context, client *xelon.Client, deviceID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		device, _, err := client.Devices.Get(ctx, deviceID)
		if err != nil {
			return nil, "", err
		}
		if device == nil {
			return nil, "", fmt.Errorf("failed to get device with id: %s", deviceID)
		}

		var deviceState string
		switch device.State {
		case 0:
			deviceState = deviceStateProvisioning
		case 1:
			deviceState = deviceStateReady
		case 2:
			deviceState = deviceStateReadyForBasicUse
		default:
			return nil, "", fmt.Errorf("failed to get correct device state: %d", device.State)
		}

		return device, deviceState, nil
	}
}
