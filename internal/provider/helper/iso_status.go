package helper

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	isoStateActive   = "active"
	isoStateInactive = "inactive"
)

func statusISOState(ctx context.Context, client *xelon.Client, isoID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		iso, resp, err := client.ISOs.Get(ctx, isoID)
		if err != nil {
			// API returns 500 sometimes for fresh created ISOs
			tflog.Info(ctx, "error by getting ISO", map[string]any{"response": resp})
			if resp != nil && resp.StatusCode == http.StatusInternalServerError {
				return iso, isoStateInactive, nil
			}
			return nil, "", err
		}
		if iso == nil {
			return nil, "", fmt.Errorf("failed to get ISO with id: %s", isoID)
		}

		var isoState string
		if iso.Active {
			isoState = isoStateActive
		} else {
			isoState = isoStateInactive
		}

		return iso, isoState, nil
	}
}
