package helper

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	loadBalancerStateProvisioning = "provisioning"
	loadBalancerStateReady        = "ready"
)

func statusLoadBalancerState(ctx context.Context, client *xelon.Client, loadBalancerID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		loadBalancer, resp, err := client.LoadBalancers.Get(ctx, loadBalancerID)
		if err != nil {
			// API returns 500 sometimes for fresh created load balancers in-provisioning state
			if resp != nil && resp.StatusCode == http.StatusInternalServerError {
				return loadBalancer, loadBalancerStateProvisioning, nil
			}
			return nil, "", err
		}
		if loadBalancer == nil {
			return nil, "", fmt.Errorf("failed to get load balancer with id: %s", loadBalancerID)
		}

		var loadBalancerState string
		switch loadBalancer.State {
		case 0, 2:
			loadBalancerState = loadBalancerStateProvisioning
		case 1:
			loadBalancerState = loadBalancerStateReady
		default:
			return nil, "", fmt.Errorf("failed to get correct load balancer state: %s", loadBalancer.HealthStatus)
		}

		return loadBalancer, loadBalancerState, nil
	}
}
