package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func WaitFirewallStateReady(ctx context.Context, client *xelon.Client, firewallID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{firewallStateProvisioning},
		Target:     []string{firewallStateReady},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusFirewallState(ctx, client, firewallID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for firewall (%s) to become ready: %w", firewallID, err)
	}
	return nil
}
