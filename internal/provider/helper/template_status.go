package helper

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	templateStateCreating = "creating"
	templateStateReady    = "ready"
)

func statusTemplateState(ctx context.Context, client *xelon.Client, templateID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		template, resp, err := client.Templates.Get(ctx, templateID)
		if err != nil {
			// API returns 500 sometimes for fresh created load balancers in-provisioning state
			if resp != nil && resp.StatusCode == http.StatusInternalServerError {
				return template, templateStateCreating, nil
			}
			return nil, "", err
		}
		if template == nil {
			return nil, "", fmt.Errorf("failed to get template with id: %s", templateID)
		}

		var templateState string
		switch template.Status {
		case 0:
			templateState = templateStateCreating
		case 1:
			templateState = templateStateReady
		default:
			return nil, "", fmt.Errorf("failed to get correct template state: %d", template.Status)
		}

		return template, templateState, nil
	}
}
