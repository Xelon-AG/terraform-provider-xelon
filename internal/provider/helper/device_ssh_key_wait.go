package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	deviceSSHKeyAssigned = "assigned"
	deviceSSHKeyAbsent   = "absent"
)

// WaitDeviceSSHKeyAssigned waits until the SSH key is listed on the device. Adding
// a key is asynchronous: the backend lists it once the guest has received it.
func WaitDeviceSSHKeyAssigned(ctx context.Context, client *xelon.Client, deviceID, sshKeyID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{deviceSSHKeyAbsent},
		Target:     []string{deviceSSHKeyAssigned},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusDeviceSSHKey(ctx, client, deviceID, sshKeyID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for ssh key (%s) to be added to device (%s): %w", sshKeyID, deviceID, err)
	}
	return nil
}

// WaitDeviceSSHKeyRemoved waits until the SSH key is no longer listed on the device.
// Removal is asynchronous: the backend unlists the key once the guest confirms it is gone.
func WaitDeviceSSHKeyRemoved(ctx context.Context, client *xelon.Client, deviceID, sshKeyID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{deviceSSHKeyAssigned},
		Target:     []string{deviceSSHKeyAbsent},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusDeviceSSHKey(ctx, client, deviceID, sshKeyID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for ssh key (%s) to be removed from device (%s): %w", sshKeyID, deviceID, err)
	}
	return nil
}

func statusDeviceSSHKey(ctx context.Context, client *xelon.Client, deviceID, sshKeyID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		sshKeys, _, err := client.Devices.ListSSHKeys(ctx, deviceID)
		if err != nil {
			return nil, "", err
		}

		if DeviceHasSSHKey(sshKeys, sshKeyID) {
			return sshKeys, deviceSSHKeyAssigned, nil
		}
		return sshKeys, deviceSSHKeyAbsent, nil
	}
}

// TemplateListsSSHKeys reports whether devices created from the template list their SSH keys.
// Only Linux devices get keys assigned; on Windows and other templates the backend never
// records the key, so waiting for it or treating it as drift would never resolve.
func TemplateListsSSHKeys(ctx context.Context, client *xelon.Client, templateID string) (bool, error) {
	template, _, err := client.Templates.Get(ctx, templateID)
	if err != nil {
		return false, err
	}
	return template.Type == templateTypeLinux, nil
}

const templateTypeLinux = "Linux"

// DeviceHasSSHKey reports whether sshKeyID is among the keys listed on a device.
func DeviceHasSSHKey(sshKeys []xelon.SSHKey, sshKeyID string) bool {
	for _, sshKey := range sshKeys {
		if sshKey.ID == sshKeyID {
			return true
		}
	}
	return false
}
