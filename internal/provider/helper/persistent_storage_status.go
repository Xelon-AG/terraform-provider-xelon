package helper

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	persistentStorageStateFormatted   = "formatted"
	persistentStorageStateUnformatted = "unformatted"

	persistentStorageStateProvisioning = "provisioning"
	persistentStorageStateReady        = "ready"
)

func statusPersistentStorageFormattedState(ctx context.Context, client *xelon.Client, persistentStorageID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		persistentStorage, resp, err := client.PersistentStorages.Get(ctx, persistentStorageID)
		if err != nil {
			// API returns 500 sometimes for fresh created persistent storages in-provisioning state
			if resp != nil && resp.StatusCode == http.StatusInternalServerError {
				return persistentStorage, persistentStorageStateUnformatted, nil
			}
			return nil, "", err
		}
		if persistentStorage == nil {
			return nil, "", fmt.Errorf("failed to get persistent storage with id: %s", persistentStorageID)
		}

		var persistentStorageFormattedState string
		if persistentStorage.Formatted {
			persistentStorageFormattedState = persistentStorageStateFormatted
		} else {
			persistentStorageFormattedState = persistentStorageStateUnformatted
		}

		return persistentStorage, persistentStorageFormattedState, nil
	}
}

func statusPersistentStorageReadyState(ctx context.Context, client *xelon.Client, persistentStorageID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		persistentStorage, resp, err := client.PersistentStorages.Get(ctx, persistentStorageID)
		if err != nil {
			// API returns 500 sometimes for fresh created persistent storages in-provisioning state
			if resp != nil && resp.StatusCode == http.StatusInternalServerError {
				return persistentStorage, persistentStorageStateProvisioning, nil
			}
			return nil, "", err
		}
		if persistentStorage == nil {
			return nil, "", fmt.Errorf("failed to get persistent storage with id: %s", persistentStorageID)
		}

		var persistentStorageReadyState string
		if persistentStorage.UUID != "" {
			persistentStorageReadyState = persistentStorageStateReady
		} else {
			persistentStorageReadyState = persistentStorageStateProvisioning
		}

		return persistentStorage, persistentStorageReadyState, nil
	}
}
