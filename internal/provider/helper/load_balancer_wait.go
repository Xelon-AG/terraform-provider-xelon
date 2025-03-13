package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func WaitLoadBalancerStateReady(ctx context.Context, client *xelon.Client, loadBalancerID string) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{loadBalancerStateProvisioning},
		Target:     []string{loadBalancerStateReady},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Delay:      3 * time.Second,
		Refresh:    statusLoadBalancerState(ctx, client, loadBalancerID),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for load balancer (%s) to become ready: %w", loadBalancerID, err)
	}
	return nil
}
