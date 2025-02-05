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
)

func statusPowerState(ctx context.Context, client *xelon.Client, deviceID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		device, resp, err := client.Devices.Get(ctx, deviceID)
		if err != nil {
			// API returns 500 for device in-provisioning state
			if resp != nil && resp.StatusCode == 503 {
				return device, devicePowerStateOff, nil
			}
			return nil, "", err
		}

		if device == nil {
			return nil, "", fmt.Errorf("failed to get device with id: %s", deviceID)
		}

		var deviceState string
		if device.PoweredOn {
			deviceState = devicePowerStateOn
		} else {
			deviceState = devicePowerStateOff
		}

		return device, deviceState, nil
	}
}
