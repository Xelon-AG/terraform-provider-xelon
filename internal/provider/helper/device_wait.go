package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func WaitDevicePowerStateOn(ctx context.Context, client *xelon.Client, deviceID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{deviceStatePowerOff},
		Target:     []string{deviceStatePowerOn},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusDevicePowerState(ctx, client, deviceID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for device (%s) to become powered on: %w", deviceID, err)
	}
	return nil
}

func WaitDevicePowerStateOff(ctx context.Context, client *xelon.Client, deviceID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{deviceStatePowerOn},
		Target:     []string{deviceStatePowerOff},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusDevicePowerState(ctx, client, deviceID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for device (%s) to become powered off: %w", deviceID, err)
	}
	return nil
}

func WaitDeviceStateReady(ctx context.Context, client *xelon.Client, deviceID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{deviceStateProvisioning, deviceStateReadyForBasicUse},
		Target:     []string{deviceStateReady},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusDeviceState(ctx, client, deviceID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for device (%s) to become ready: %w", deviceID, err)
	}
	return nil
}
