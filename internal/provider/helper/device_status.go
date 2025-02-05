package helper

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	devicePowerStateOn  = "poweredOn"
	devicePowerStateOff = "poweredOff"

	deviceStateProvisioning     = "provisioning"
	deviceStateReadyForBasicUse = "readyForBasicUse"
	deviceStateReady            = "ready"
)

func statusPowerState(ctx context.Context, client *xelon.Client, deviceID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		device, _, err := client.Devices.Get(ctx, deviceID)
		if err != nil {
			return nil, "", err
		}
		if device == nil {
			return nil, "", fmt.Errorf("failed to get device with id: %s", deviceID)
		}

		var devicePowerState string
		if device.PoweredOn {
			devicePowerState = devicePowerStateOn
		} else {
			devicePowerState = devicePowerStateOff
		}

		return device, devicePowerState, nil
	}
}

func statusState(ctx context.Context, client *xelon.Client, deviceID string) retry.StateRefreshFunc {
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
