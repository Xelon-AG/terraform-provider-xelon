package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func WaitISOStateReady(ctx context.Context, client *xelon.Client, isoID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{isoStateCreating},
		Target:     []string{isoStateReady},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusISOState(ctx, client, isoID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for ISO (%s) to become active: %w", isoID, err)
	}
	return nil
}
