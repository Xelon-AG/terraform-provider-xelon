package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ssh_key_id is changed in place (add the new key, remove the old one), so no
// change to it may plan a replacement of the device.
func TestResourceXelonDevice_Update_SSHKeyChangeDoesNotRequireReplacement(t *testing.T) {
	tests := map[string]struct {
		configValue types.String
		stateValue  types.String
	}{
		"key changed": {configValue: types.StringValue("ssh-key-2"), stateValue: types.StringValue("ssh-key-1")},
		"key added":   {configValue: types.StringValue("ssh-key-1"), stateValue: types.StringNull()},
		"key removed": {configValue: types.StringNull(), stateValue: types.StringValue("ssh-key-1")},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			deviceSchema := testDeviceResourceSchema(t)

			sshKeyID, ok := deviceSchema.Attributes["ssh_key_id"].(schema.StringAttribute)
			require.True(t, ok)

			plan := testDeviceResourcePlan(t, ctx, deviceSchema, types.StringNull(), types.StringNull())
			state := testDeviceResourcePlan(t, ctx, deviceSchema, types.StringNull(), types.StringNull())

			response := &planmodifier.StringResponse{PlanValue: test.configValue}
			for _, modifier := range sshKeyID.PlanModifiers {
				modifier.PlanModifyString(ctx, planmodifier.StringRequest{
					ConfigValue: test.configValue,
					Plan:        plan,
					PlanValue:   test.configValue,
					State:       tfsdk.State{Schema: deviceSchema, Raw: state.Raw},
					StateValue:  test.stateValue,
				}, response)
			}

			require.False(t, response.Diagnostics.HasError())
			assert.False(t, response.RequiresReplace)
			assert.True(t, response.PlanValue.Equal(test.configValue))
		})
	}
}
