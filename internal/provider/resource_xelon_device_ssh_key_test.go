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

func TestResourceXelonDevice_Schema_SSHKeyIDs(t *testing.T) {
	deviceSchema := testDeviceResourceSchema(t)

	sshKeyIDs, ok := deviceSchema.Attributes["ssh_key_ids"].(schema.SetAttribute)
	require.True(t, ok)
	assert.True(t, sshKeyIDs.Optional)
	assert.True(t, sshKeyIDs.Computed)
	assert.Equal(t, types.StringType, sshKeyIDs.ElementType)

	sshKeyID, ok := deviceSchema.Attributes["ssh_key_id"].(schema.StringAttribute)
	require.True(t, ok)
	assert.NotEmpty(t, sshKeyID.DeprecationMessage)
	require.Len(t, sshKeyID.Validators, 1)
	assert.Contains(t, sshKeyID.Validators[0].Description(context.Background()), "ssh_key_ids")
}

func TestResourceXelonDevice_SSHKeyChanges(t *testing.T) {
	tests := map[string]struct {
		current, desired []string
		toAdd, toRemove  []string
	}{
		"unchanged":        {current: []string{"a", "b"}, desired: []string{"b", "a"}},
		"key added":        {current: []string{"a"}, desired: []string{"a", "b"}, toAdd: []string{"b"}},
		"key removed":      {current: []string{"a", "b"}, desired: []string{"a"}, toRemove: []string{"b"}},
		"key swapped":      {current: []string{"a"}, desired: []string{"b"}, toAdd: []string{"b"}, toRemove: []string{"a"}},
		"all keys removed": {current: []string{"a", "b"}, desired: nil, toRemove: []string{"a", "b"}},
		"first keys added": {current: nil, desired: []string{"a", "b"}, toAdd: []string{"a", "b"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			toAdd, toRemove := sshKeyChanges(test.current, test.desired)

			assert.Equal(t, test.toAdd, toAdd)
			assert.Equal(t, test.toRemove, toRemove)
		})
	}
}
