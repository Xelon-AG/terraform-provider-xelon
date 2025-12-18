package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func WaitPersistentStorageStateFormatted(ctx context.Context, client *xelon.Client, persistentStorageID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{persistentStorageStateUnformatted},
		Target:     []string{persistentStorageStateFormatted},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusPersistentStorageFormattedState(ctx, client, persistentStorageID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for persistent storage (%s) to become formatted: %w", persistentStorageID, err)
	}
	return nil
}

func WaitPersistentStorageStateReady(ctx context.Context, client *xelon.Client, persistentStorageID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{persistentStorageStateProvisioning},
		Target:     []string{persistentStorageStateReady},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusPersistentStorageReadyState(ctx, client, persistentStorageID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for persistent storage (%s) to become ready: %w", persistentStorageID, err)
	}
	return nil
}
