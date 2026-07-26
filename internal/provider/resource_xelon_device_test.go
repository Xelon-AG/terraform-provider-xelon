package provider

import (
	"context"
	"net/netip"
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

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
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

func TestResourceXelonDevice_Schema_NetworkIPv4Address(t *testing.T) {
	deviceSchema := testDeviceResourceSchema(t)

	networks, ok := deviceSchema.Attributes["networks"].(schema.SetNestedAttribute)
	require.True(t, ok)

	ipv4Address, ok := networks.NestedObject.Attributes["ipv4_address"].(schema.StringAttribute)
	require.True(t, ok)

	assert.True(t, ipv4Address.Optional)
	assert.True(t, ipv4Address.Computed)
	assert.Empty(t, ipv4Address.PlanModifiers)
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

func TestResourceXelonDevice_NetworkIPv4Addresses_OmittedIPBackendIPv4Stored(t *testing.T) {
	networks := populateDeviceNetworkIPv4Addresses(
		[]deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringNull()),
		},
		[]xelon.DeviceNetwork{
			testDeviceNetworkInfo("network-id", true, "10.0.0.25"),
		},
	)

	require.Len(t, networks, 1)
	assert.Equal(t, "10.0.0.25", networks[0].IPAddress.ValueString())
}

func TestResourceXelonDevice_NetworkIPv4Addresses_ManualIPBackendIPv4Stored(t *testing.T) {
	networks := populateDeviceNetworkIPv4Addresses(
		[]deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringValue("10.0.0.50")),
		},
		[]xelon.DeviceNetwork{
			testDeviceNetworkInfo("network-id", true, "10.0.0.50"),
		},
	)

	require.Len(t, networks, 1)
	assert.Equal(t, "10.0.0.50", networks[0].IPAddress.ValueString())
}

func TestResourceXelonDevice_NetworkIPv4Addresses_BackendIPv4ReplacesStaleState(t *testing.T) {
	networks := populateDeviceNetworkIPv4Addresses(
		[]deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringValue("10.0.0.10")),
		},
		[]xelon.DeviceNetwork{
			testDeviceNetworkInfo("network-id", true, "10.0.0.25"),
		},
	)

	require.Len(t, networks, 1)
	assert.Equal(t, "10.0.0.25", networks[0].IPAddress.ValueString())
}

func TestResourceXelonDevice_NetworkIPv4Addresses_MissingBackendIPv4ClearsStaleState(t *testing.T) {
	networks := populateDeviceNetworkIPv4Addresses(
		[]deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringValue("10.0.0.10")),
		},
		[]xelon.DeviceNetwork{
			testDeviceNetworkInfo("network-id", true),
		},
	)

	require.Len(t, networks, 1)
	assert.True(t, networks[0].IPAddress.IsNull())
}

func TestResourceXelonDevice_NetworkIPv4Addresses_BackendIPv6BeforeIPv4SelectsIPv4(t *testing.T) {
	networks := populateDeviceNetworkIPv4Addresses(
		[]deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringNull()),
		},
		[]xelon.DeviceNetwork{
			testDeviceNetworkInfo("network-id", true, "2001:db8::1", "10.0.0.25"),
		},
	)

	require.Len(t, networks, 1)
	assert.Equal(t, "10.0.0.25", networks[0].IPAddress.ValueString())
}

func TestResourceXelonDevice_NetworkIPv4Addresses_MatchingNetworkDisconnectedIgnored(t *testing.T) {
	networks := populateDeviceNetworkIPv4Addresses(
		[]deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringValue("10.0.0.25")),
		},
		[]xelon.DeviceNetwork{
			testDeviceNetworkInfo("network-id", false, "10.0.0.30"),
		},
	)

	require.Len(t, networks, 1)
	assert.True(t, networks[0].IPAddress.IsNull())
}

func TestResourceXelonDevice_NetworkIPv4Addresses_MultipleNetworksMatchedByIDNotOrder(t *testing.T) {
	networks := populateDeviceNetworkIPv4Addresses(
		[]deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-a", types.StringNull()),
			testDeviceNetworkResourceModel("network-b", types.StringValue("10.0.0.20")),
		},
		[]xelon.DeviceNetwork{
			testDeviceNetworkInfo("network-b", true, "10.0.0.20"),
			testDeviceNetworkInfo("network-a", true, "10.0.0.10"),
		},
	)

	require.Len(t, networks, 2)
	assert.Equal(t, "10.0.0.10", networks[0].IPAddress.ValueString())
	assert.Equal(t, "10.0.0.20", networks[1].IPAddress.ValueString())
}

func TestResourceXelonDevice_NetworkIPv4Addresses_MatchingNetworkWithoutIPv4DoesNotBlockLaterMatch(t *testing.T) {
	networks := populateDeviceNetworkIPv4Addresses(
		[]deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringNull()),
		},
		[]xelon.DeviceNetwork{
			testDeviceNetworkInfo("network-id", true, "2001:db8::1"),
			testDeviceNetworkInfo("network-id", true, "10.0.0.25"),
		},
	)

	require.Len(t, networks, 1)
	assert.Equal(t, "10.0.0.25", networks[0].IPAddress.ValueString())
}

func TestResourceXelonDevice_Model_FromAPI_PreservesCreateTimeOnlyFields(t *testing.T) {
	ctx := context.Background()
	device := &xelon.Device{
		CPUCores:              4,
		CPUCoresHotAddEnabled: true,
		DisplayName:           "backend-device",
		HostName:              "backend-hostname",
		ID:                    "device-id",
		RAM:                   8,
		RAMHotAddEnabled:      true,
		Storages: []xelon.DeviceStorage{
			{ID: "disk-id", Size: 20},
			{ID: "swap-disk-id", Size: 2},
		},
	}
	expected := deviceResourceModel{
		CPUCoreCount:     types.Int64Value(4),
		CPUCoreHotPlug:   types.BoolValue(true),
		DiskID:           types.StringValue("disk-id"),
		DiskSize:         types.Int64Value(20),
		DisplayName:      types.StringValue("backend-device"),
		EnableMonitoring: types.BoolValue(false),
		Hostname:         types.StringValue("backend-hostname"),
		ID:               types.StringValue("device-id"),
		Memory:           types.Int64Value(8),
		MemoryHotPlug:    types.BoolValue(true),
		Networks: []deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringValue("10.0.0.25")),
		},
		Password:     types.StringValue("password"),
		SendEmail:    types.BoolValue(true),
		SSHKeyID:     types.StringValue("ssh-key-id"),
		ScriptID:     types.StringValue("script-id"),
		SwapDiskID:   types.StringValue("swap-disk-id"),
		SwapDiskSize: types.Int64Value(2),
		TemplateID:   types.StringValue("template-id"),
		TenantID:     types.StringValue("tenant-id"),
		UserData:     types.StringValue("user-data"),
	}

	actual := deviceResourceModel{
		DiskSize:     types.Int64Value(20),
		Networks:     []deviceNetworkResourceModel{testDeviceNetworkResourceModel("network-id", types.StringUnknown())},
		Password:     types.StringValue("password"),
		SendEmail:    types.BoolValue(true),
		SSHKeyID:     types.StringValue("ssh-key-id"),
		ScriptID:     types.StringValue("script-id"),
		SwapDiskSize: types.Int64Value(2),
		TemplateID:   types.StringValue("template-id"),
		TenantID:     types.StringValue("tenant-id"),
		UserData:     types.StringValue("user-data"),
	}

	actual.fromAPI(ctx, device, []xelon.DeviceNetwork{
		testDeviceNetworkInfo("network-id", true, "10.0.0.25"),
	})

	assert.Equal(t, expected, actual)
}

func TestResourceXelonDevice_Model_FromAPI_PopulatesComputedNetworkIPv4(t *testing.T) {
	expected := []deviceNetworkResourceModel{
		testDeviceNetworkResourceModel("network-id", types.StringValue("10.0.0.25")),
	}

	actual := deviceResourceModel{
		DiskSize:     types.Int64Value(20),
		SwapDiskSize: types.Int64Value(2),
		Networks: []deviceNetworkResourceModel{
			testDeviceNetworkResourceModel("network-id", types.StringUnknown()),
		},
	}
	device := &xelon.Device{
		Storages: []xelon.DeviceStorage{
			{ID: "disk-id", Size: 20},
			{ID: "swap-disk-id", Size: 2},
		},
	}

	actual.fromAPI(context.Background(), device, []xelon.DeviceNetwork{
		testDeviceNetworkInfo("network-id", true, "10.0.0.25"),
	})

	assert.Equal(t, expected, actual.Networks)
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

func testDeviceNetworkResourceModel(networkID string, ipAddress types.String) deviceNetworkResourceModel {
	return deviceNetworkResourceModel{
		Connected:   types.BoolValue(true),
		ID:          types.StringValue(networkID),
		IPAddress:   ipAddress,
		IPAddressID: types.StringNull(),
	}
}

func testDeviceNetworkInfo(networkID string, connected bool, ipAddresses ...string) xelon.DeviceNetwork {
	parsedIPAddresses := make(xelon.DeviceNetworkIPAddresses, 0, len(ipAddresses))
	for _, ipAddress := range ipAddresses {
		parsedIPAddresses = append(parsedIPAddresses, netip.MustParseAddr(ipAddress))
	}

	return xelon.DeviceNetwork{
		Connected:   connected,
		ID:          networkID,
		IPAddresses: parsedIPAddresses,
	}
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
