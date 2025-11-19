package helper

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	firewallStateProvisioning = "provisioning"
	firewallStateReady        = "ready"
)

func statusFirewallState(ctx context.Context, client *xelon.Client, firewallID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		firewall, resp, err := client.Firewalls.Get(ctx, firewallID)
		if err != nil {
			// API returns 500 sometimes for fresh created firewalls in-provisioning state
			if resp != nil && resp.StatusCode == http.StatusInternalServerError {
				return firewall, firewallStateProvisioning, nil
			}
			return nil, "", err
		}
		if firewall == nil {
			return nil, "", fmt.Errorf("failed to get firewall with id: %s", firewallID)
		}

		var firewallState string
		switch firewall.State {
		case 0, 2:
			firewallState = firewallStateProvisioning
		case 1:
			firewallState = firewallStateReady
		default:
			return nil, "", fmt.Errorf("failed to get correct firewall state: %s", firewall.HealthStatus)
		}

		return firewall, firewallState, nil
	}
}
