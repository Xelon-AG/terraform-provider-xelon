package storage

import (
	"context"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	persistentStorageStateCreated    = "Created"
	persistentStorageStateInProgress = "InProgress"
)

func statusStorageState(ctx context.Context, client *xelon.Client, tenantID, localID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		storage, _, err := client.PersistentStorages.Get(ctx, tenantID, localID)
		if err != nil {
			return nil, "", err
		}

		state := persistentStorageStateInProgress
		if storage.Formatted == 1 && storage.UUID != "" {
			state = persistentStorageStateCreated
		}

		return storage, state, nil
	}
}
