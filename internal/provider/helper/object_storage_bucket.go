package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	objectStorageBucketVersioningDisabled = "versioningDisabled"
	objectStorageBucketVersioningEnabled  = "versioningEnabled"
	objectStorageBucketVersioningPending  = "versioningPending"
)

func WaitObjectStorageBucketVersioningDisabled(ctx context.Context, client *xelon.Client, bucketName, userID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{objectStorageBucketVersioningPending, objectStorageBucketVersioningEnabled},
		Target:     []string{objectStorageBucketVersioningDisabled},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    stateObjectStorageBucketVersioning(ctx, client, bucketName, userID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for object storage bucket versioning (%s) to become disabled: %w", bucketName, err)
	}
	return nil
}

func WaitObjectStorageBucketVersioningEnabled(ctx context.Context, client *xelon.Client, bucketName, userID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{objectStorageBucketVersioningPending, objectStorageBucketVersioningDisabled},
		Target:     []string{objectStorageBucketVersioningEnabled},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    stateObjectStorageBucketVersioning(ctx, client, bucketName, userID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for object storage bucket versioning (%s) to become enabled: %w", bucketName, err)
	}
	return nil
}

func stateObjectStorageBucketVersioning(ctx context.Context, client *xelon.Client, bucketName, userID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		bucket, _, err := client.ObjectStorages.GetBucket(ctx, bucketName, userID)
		if err != nil {
			return nil, "", err
		}
		if bucket == nil {
			return nil, "", fmt.Errorf("failed to get object storage bucket with name: %s", bucketName)
		}

		if bucket.VersioningStatus == xelon.ObjectStorageBucketVersioningStatusPending {
			return bucket, objectStorageBucketVersioningPending, nil
		}
		if bucket.VersioningStatus != xelon.ObjectStorageBucketVersioningStatusDone {
			return nil, "", fmt.Errorf("unexpected object storage bucket versioning status: %s", bucket.VersioningStatus)
		}

		if bucket.VersioningEnabled {
			return bucket, objectStorageBucketVersioningEnabled, nil
		}
		return bucket, objectStorageBucketVersioningDisabled, nil
	}
}
