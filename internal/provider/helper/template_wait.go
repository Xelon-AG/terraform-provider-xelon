package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func WaitTemplateStateReady(ctx context.Context, client *xelon.Client, templateID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{templateStateCreating},
		Target:     []string{templateStateReady},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusTemplateState(ctx, client, templateID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for template (%s) to become ready: %w", templateID, err)
	}
	return nil
}
