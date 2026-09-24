package helper

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	deviceDiskSSHKeyAssigned   = "sshKeyAssigned"
	deviceDiskSSHKeyUnassigned = "sshKeyUnassigned"
)

func WaitDeviceSSHKeyAssigned(ctx context.Context, client *xelon.Client, deviceID, sshKeyID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{deviceDiskSSHKeyUnassigned},
		Target:     []string{deviceDiskSSHKeyAssigned},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    stateDeviceSSHKeyStatus(ctx, client, deviceID, sshKeyID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for SSH key (%s) to be assigned to device (%s): %w", sshKeyID, deviceID, err)
	}
	return nil
}

func WaitDeviceSSHKeyUnassigned(ctx context.Context, client *xelon.Client, deviceID, sshKeyID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{deviceDiskSSHKeyAssigned},
		Target:     []string{deviceDiskSSHKeyUnassigned},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    stateDeviceSSHKeyStatus(ctx, client, deviceID, sshKeyID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for SSH key (%s) to be unassigned from device (%s): %w", sshKeyID, deviceID, err)
	}
	return nil
}

func stateDeviceSSHKeyStatus(ctx context.Context, client *xelon.Client, deviceID, sshKeyID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		sshKeys, _, err := client.Devices.ListSSHKeys(ctx, deviceID)
		if err != nil {
			return nil, "", err
		}

		if slices.ContainsFunc(sshKeys, func(sshKey xelon.SSHKey) bool { return sshKey.ID == sshKeyID }) {
			return sshKeys, deviceDiskSSHKeyAssigned, nil
		}
		return sshKeys, deviceDiskSSHKeyUnassigned, nil
	}
}
