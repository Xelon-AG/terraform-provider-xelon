package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func WaitPowerStateOn(ctx context.Context, client *xelon.Client, deviceID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{devicePowerStateOff, ""},
		Target:     []string{devicePowerStateOn},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    statusPowerState(ctx, client, deviceID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for device (%s) to become PoweredOn: %w", deviceID, err)
	}
	return nil
}
