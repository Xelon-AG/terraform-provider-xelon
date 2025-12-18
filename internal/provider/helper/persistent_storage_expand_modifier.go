package helper

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// expandOnlyStorageSizeModifier implements the plan modifier.
type expandOnlyStorageSizeModifier struct{}

// ExpandOnlyStorageSizeModifier returns a plan modifier that ensures a disk's size
// can only be increased by comparing known and planned values.
func ExpandOnlyStorageSizeModifier() planmodifier.Int64 {
	return expandOnlyStorageSizeModifier{}
}

func (m expandOnlyStorageSizeModifier) Description(_ context.Context) string {
	return "Ensures that storage size can only be increased, not decreased."
}

func (m expandOnlyStorageSizeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m expandOnlyStorageSizeModifier) PlanModifyInt64(_ context.Context, request planmodifier.Int64Request, response *planmodifier.Int64Response) {
	// do nothing if there is no state (resource is being created)
	if request.State.Raw.IsNull() {
		return
	}
	// do nothing on resource destroy
	if request.Plan.Raw.IsNull() {
		return
	}
	// do nothing if the plan and state values are equal
	if request.PlanValue.Equal(request.StateValue) {
		return
	}

	stateSize := request.StateValue.ValueInt64()
	planSize := request.PlanValue.ValueInt64()
	if planSize < stateSize {
		response.Diagnostics.AddAttributeError(
			request.Path,
			"Invalid Attribute Value",
			fmt.Sprintf(
				"The storage size cannot be decreased from %d to %d. This operation is blocked to accidental prevent data loss.",
				stateSize,
				planSize,
			),
		)
	}
}
