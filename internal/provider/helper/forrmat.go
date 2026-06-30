package helper

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func FormatTimeRFC3339(t *time.Time) types.String {
	if t == nil {
		return types.StringNull()
	}
	return types.StringValue(t.UTC().Format(time.RFC3339))
}
