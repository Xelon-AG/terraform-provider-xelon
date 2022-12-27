package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func WaitStorageStateCreated(ctx context.Context, client *xelon.Client, tenantID, localID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{persistentStorageStateInProgress},
		Target:  []string{persistentStorageStateCreated},
		Timeout: 10 * time.Minute,
		Delay:   5 * time.Second,
		Refresh: statusStorageState(ctx, client, tenantID, localID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for persistent storage (%s) to become Created: %w", localID, err)
	}

	return nil
}
