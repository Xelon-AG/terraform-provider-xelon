package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceXelonDevice_Schema_Password(t *testing.T) {
	deviceSchema := testDeviceResourceSchema(t)

	password, ok := deviceSchema.Attributes["password"].(schema.StringAttribute)
	require.True(t, ok)

	assert.True(t, password.Optional)
	assert.False(t, password.Computed)
	assert.True(t, password.Sensitive)
	require.Len(t, password.PlanModifiers, 1)
	assert.Contains(t, password.PlanModifiers[0].Description(context.Background()), "configured and changes")
}

func TestResourceXelonDevice_Create_RequiresPasswordOrUserData(t *testing.T) {
	ctx := context.Background()
	deviceSchema := testDeviceResourceSchema(t)
	plan := testDeviceResourcePlan(t, ctx, deviceSchema, types.StringNull(), types.StringNull())
	state := tfsdk.State{
		Schema: deviceSchema,
		Raw:    tftypes.NewValue(deviceSchema.Type().TerraformType(ctx), nil),
	}

	response := &resource.ModifyPlanResponse{}
	NewDeviceResource().(*deviceResource).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:  plan,
		State: state,
	}, response)

	require.True(t, response.Diagnostics.HasError())
	assert.True(t, response.Diagnostics.Equal(expectedDeviceMissingPasswordOrUserDataDiagnostics()))
}

func TestResourceXelonDevice_Import_DoesNotRequirePasswordOrUserData(t *testing.T) {
	ctx := context.Background()
	deviceSchema := testDeviceResourceSchema(t)
	plan := testDeviceResourcePlan(t, ctx, deviceSchema, types.StringNull(), types.StringNull())
	state := tfsdk.State{
		Schema: deviceSchema,
		Raw:    plan.Raw,
	}

	response := &resource.ModifyPlanResponse{}
	NewDeviceResource().(*deviceResource).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:  plan,
		State: state,
	}, response)

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonDevice_Import_PasswordOmittedDoesNotRequireReplacement(t *testing.T) {
	response := testDevicePasswordPlanModifierResponse(t, types.StringNull(), types.StringNull(), types.StringNull())

	require.False(t, response.Diagnostics.HasError())
	assert.False(t, response.RequiresReplace)
}

func TestResourceXelonDevice_Import_ConfiguredPasswordRequiresReplacement(t *testing.T) {
	response := testDevicePasswordPlanModifierResponse(t, types.StringValue("new-password"), types.StringValue("new-password"), types.StringNull())

	require.False(t, response.Diagnostics.HasError())
	assert.True(t, response.RequiresReplace)
}

func TestResourceXelonDevice_Update_PasswordChangeRequiresReplacement(t *testing.T) {
	response := testDevicePasswordPlanModifierResponse(t, types.StringValue("new-password"), types.StringValue("new-password"), types.StringValue("old-password"))

	require.False(t, response.Diagnostics.HasError())
	assert.True(t, response.RequiresReplace)
}

func TestResourceXelonDevice_Replacement_RequiresPasswordOrUserData(t *testing.T) {
	ctx := context.Background()
	deviceSchema := testDeviceResourceSchema(t)
	plan := testDeviceResourcePlan(t, ctx, deviceSchema, types.StringNull(), types.StringNull())
	statePlan := testDeviceResourcePlanWithTemplateID(t, ctx, deviceSchema, types.StringNull(), types.StringNull(), "old-template-id")
	state := tfsdk.State{
		Schema: deviceSchema,
		Raw:    statePlan.Raw,
	}

	response := &resource.ModifyPlanResponse{}
	NewDeviceResource().(*deviceResource).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:  plan,
		State: state,
	}, response)

	require.True(t, response.Diagnostics.HasError())
	assert.True(t, response.Diagnostics.Equal(expectedDeviceMissingPasswordOrUserDataDiagnostics()))
}

func testDeviceResourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	r := NewDeviceResource()
	response := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, response)
	require.False(t, response.Diagnostics.HasError())

	return response.Schema
}

func testDeviceResourcePlan(t *testing.T, ctx context.Context, deviceSchema schema.Schema, password, userData types.String) tfsdk.Plan {
	t.Helper()

	return testDeviceResourcePlanWithTemplateID(t, ctx, deviceSchema, password, userData, "template-id")
}

func testDeviceResourcePlanWithTemplateID(t *testing.T, ctx context.Context, deviceSchema schema.Schema, password, userData types.String, templateID string) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: deviceSchema}
	diags := plan.Set(ctx, &deviceResourceModel{
		BackupJobID:      types.Int64Null(),
		CPUCoreCount:     types.Int64Value(2),
		CPUCoreHotPlug:   types.BoolNull(),
		DiskID:           types.StringUnknown(),
		DiskSize:         types.Int64Value(10),
		DisplayName:      types.StringValue("test-device"),
		EnableMonitoring: types.BoolNull(),
		Hostname:         types.StringValue("test-device"),
		ID:               types.StringUnknown(),
		Memory:           types.Int64Value(2),
		MemoryHotPlug:    types.BoolNull(),
		Networks: []deviceNetworkResourceModel{
			{
				Connected:   types.BoolValue(true),
				ID:          types.StringValue("network-id"),
				IPAddress:   types.StringNull(),
				IPAddressID: types.StringNull(),
			},
		},
		Password:     password,
		SendEmail:    types.BoolNull(),
		SSHKeyID:     types.StringNull(),
		ScriptID:     types.StringNull(),
		SwapDiskID:   types.StringUnknown(),
		SwapDiskSize: types.Int64Value(1),
		TemplateID:   types.StringValue(templateID),
		TenantID:     types.StringValue("tenant-id"),
		UserData:     userData,
	})
	require.False(t, diags.HasError())

	return plan
}

func testDevicePasswordPlanModifierResponse(t *testing.T, configValue, planValue, stateValue types.String) *planmodifier.StringResponse {
	t.Helper()

	ctx := context.Background()
	deviceSchema := testDeviceResourceSchema(t)

	password, ok := deviceSchema.Attributes["password"].(schema.StringAttribute)
	require.True(t, ok)
	require.Len(t, password.PlanModifiers, 1)

	plan := testDeviceResourcePlan(t, ctx, deviceSchema, planValue, types.StringNull())
	statePlan := testDeviceResourcePlan(t, ctx, deviceSchema, stateValue, types.StringNull())

	response := &planmodifier.StringResponse{
		PlanValue: planValue,
	}
	password.PlanModifiers[0].PlanModifyString(ctx, planmodifier.StringRequest{
		ConfigValue: configValue,
		Plan:        plan,
		PlanValue:   planValue,
		State: tfsdk.State{
			Schema: deviceSchema,
			Raw:    statePlan.Raw,
		},
		StateValue: stateValue,
	}, response)

	return response
}

func expectedDeviceMissingPasswordOrUserDataDiagnostics() diag.Diagnostics {
	return diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(
			path.Root("password"),
			"Missing password or user_data",
			`Either "password" or "user_data" must be specified when creating or replacing a device.`,
		),
	}
}
