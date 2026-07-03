package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceXelonKubernetesCluster_ModifyPlan_CreateControlPlaneHADisabled(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA: types.BoolValue(false),
		planLoadBalancerHA: types.BoolValue(true),
		stateIsNull:        true,
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonKubernetesCluster_ModifyPlan_CreateLoadBalancerHADisabled(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA: types.BoolValue(true),
		planLoadBalancerHA: types.BoolValue(false),
		stateIsNull:        true,
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonKubernetesCluster_ModifyPlan_ControlPlaneHADowngrade(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(false),
		stateControlPlaneHA: types.BoolValue(true),
		planLoadBalancerHA:  types.BoolValue(true),
		stateLoadBalancerHA: types.BoolValue(true),
	})

	require.True(t, response.Diagnostics.HasError())
	assert.True(t, response.Diagnostics.Equal(expectedKubernetesClusterControlPlaneHADowngradeDiagnostics()))
}

func TestResourceXelonKubernetesCluster_ModifyPlan_LoadBalancerHADowngrade(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(true),
		stateControlPlaneHA: types.BoolValue(true),
		planLoadBalancerHA:  types.BoolValue(false),
		stateLoadBalancerHA: types.BoolValue(true),
	})

	require.True(t, response.Diagnostics.HasError())
	assert.True(t, response.Diagnostics.Equal(expectedKubernetesClusterLoadBalancerHADowngradeDiagnostics()))
}

func TestResourceXelonKubernetesCluster_ModifyPlan_ControlPlaneHAUpgrade_LoadBalancerAlreadyHA(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(true),
		stateControlPlaneHA: types.BoolValue(false),
		planLoadBalancerHA:  types.BoolValue(true),
		stateLoadBalancerHA: types.BoolValue(true),
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonKubernetesCluster_ModifyPlan_ControlPlaneHAPartialUpgrade(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(true),
		stateControlPlaneHA: types.BoolValue(false),
		planLoadBalancerHA:  types.BoolValue(false),
		stateLoadBalancerHA: types.BoolValue(false),
	})

	require.True(t, response.Diagnostics.HasError())
	assert.True(t, response.Diagnostics.Equal(expectedKubernetesClusterControlPlaneHAPartialUpgradeDiagnostics()))
}

func TestResourceXelonKubernetesCluster_ModifyPlan_LoadBalancerHAUpgrade_ControlPlaneAlreadyHA(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(true),
		stateControlPlaneHA: types.BoolValue(true),
		planLoadBalancerHA:  types.BoolValue(true),
		stateLoadBalancerHA: types.BoolValue(false),
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonKubernetesCluster_ModifyPlan_LoadBalancerHAPartialUpgrade(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(false),
		stateControlPlaneHA: types.BoolValue(false),
		planLoadBalancerHA:  types.BoolValue(true),
		stateLoadBalancerHA: types.BoolValue(false),
	})

	require.True(t, response.Diagnostics.HasError())
	assert.True(t, response.Diagnostics.Equal(expectedKubernetesClusterLoadBalancerHAPartialUpgradeDiagnostics()))
}

func TestResourceXelonKubernetesCluster_ModifyPlan_FullHAUpgrade(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(true),
		stateControlPlaneHA: types.BoolValue(false),
		planLoadBalancerHA:  types.BoolValue(true),
		stateLoadBalancerHA: types.BoolValue(false),
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonKubernetesCluster_ModifyPlan_HAUnchanged(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(true),
		stateControlPlaneHA: types.BoolValue(true),
		planLoadBalancerHA:  types.BoolValue(false),
		stateLoadBalancerHA: types.BoolValue(false),
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonKubernetesCluster_ModifyPlan_ControlPlaneHAUnknownPlanAllowed(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolUnknown(),
		stateControlPlaneHA: types.BoolValue(false),
		planLoadBalancerHA:  types.BoolValue(false),
		stateLoadBalancerHA: types.BoolValue(false),
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonKubernetesCluster_ModifyPlan_LoadBalancerHAUnknownPlanAllowed(t *testing.T) {
	response := testKubernetesClusterModifyPlanResponse(t, kubernetesClusterModifyPlanCase{
		planControlPlaneHA:  types.BoolValue(false),
		stateControlPlaneHA: types.BoolValue(false),
		planLoadBalancerHA:  types.BoolUnknown(),
		stateLoadBalancerHA: types.BoolValue(false),
	})

	assert.False(t, response.Diagnostics.HasError())
}

type kubernetesClusterModifyPlanCase struct {
	planControlPlaneHA  types.Bool
	planLoadBalancerHA  types.Bool
	stateControlPlaneHA types.Bool
	stateLoadBalancerHA types.Bool
	stateIsNull         bool
}

type kubernetesClusterPlanValues struct {
	controlPlaneHA types.Bool
	loadBalancerHA types.Bool
	id             types.String
}

func testKubernetesClusterModifyPlanResponse(t *testing.T, testCase kubernetesClusterModifyPlanCase) *resource.ModifyPlanResponse {
	t.Helper()

	ctx := context.Background()
	kubernetesClusterSchema := testKubernetesClusterResourceSchema(t)

	plan := testKubernetesClusterResourcePlan(t, ctx, kubernetesClusterSchema, kubernetesClusterPlanValues{
		controlPlaneHA: testCase.planControlPlaneHA,
		loadBalancerHA: testCase.planLoadBalancerHA,
		id:             types.StringUnknown(),
	})

	state := tfsdk.State{
		Schema: kubernetesClusterSchema,
		Raw:    tftypes.NewValue(kubernetesClusterSchema.Type().TerraformType(ctx), nil),
	}
	if !testCase.stateIsNull {
		statePlan := testKubernetesClusterResourcePlan(t, ctx, kubernetesClusterSchema, kubernetesClusterPlanValues{
			controlPlaneHA: testCase.stateControlPlaneHA,
			loadBalancerHA: testCase.stateLoadBalancerHA,
			id:             types.StringValue("kubernetes-cluster-id"),
		})
		state.Raw = statePlan.Raw
	}

	response := &resource.ModifyPlanResponse{}
	NewKubernetesClusterResource().(*kubernetesClusterResource).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:  plan,
		State: state,
	}, response)

	return response
}

func testKubernetesClusterResourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	r := NewKubernetesClusterResource()
	response := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, response)
	require.False(t, response.Diagnostics.HasError())

	return response.Schema
}

func testKubernetesClusterResourcePlan(t *testing.T, ctx context.Context, kubernetesClusterSchema schema.Schema, values kubernetesClusterPlanValues) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: kubernetesClusterSchema}
	diags := plan.Set(ctx, &kubernetesClusterResourceModel{
		CloudID:           types.StringValue("cloud-id"),
		ControlPlane:      testKubernetesClusterNodeSpecObject(values.controlPlaneHA),
		ID:                values.id,
		KubernetesVersion: types.StringValue("1.31.0"),
		LoadBalancer:      testKubernetesClusterNodeSpecObject(values.loadBalancerHA),
		Name:              types.StringValue("test-cluster"),
		TalosVersion:      types.StringValue("1.9.0"),
		TenantID:          types.StringValue("tenant-id"),
		Timeouts: timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"update": types.StringType,
		})},
	})
	require.False(t, diags.HasError())

	return plan
}

func testKubernetesClusterNodeSpecObject(highAvailabilityEnabled types.Bool) types.Object {
	return types.ObjectValueMust(kubernetesClusterNodeSpecAttributeTypes(), map[string]attr.Value{
		"cpu_core_count":            types.Int64Value(2),
		"disk_size":                 types.Int64Value(50),
		"high_availability_enabled": highAvailabilityEnabled,
		"memory":                    types.Int64Value(4),
	})
}

func expectedKubernetesClusterControlPlaneHADowngradeDiagnostics() diag.Diagnostics {
	return diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(
			path.Root("control_plane").AtName("high_availability_enabled"),
			"Downgrading control plane high availability is not supported",
			"High availability cannot be changed from true to false for an existing Kubernetes cluster.",
		),
	}
}

func expectedKubernetesClusterLoadBalancerHADowngradeDiagnostics() diag.Diagnostics {
	return diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(
			path.Root("load_balancer").AtName("high_availability_enabled"),
			"Downgrading load balancer high availability is not supported",
			"High availability cannot be changed from true to false for an existing Kubernetes cluster.",
		),
	}
}

func expectedKubernetesClusterControlPlaneHAPartialUpgradeDiagnostics() diag.Diagnostics {
	return diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(
			path.Root("control_plane").AtName("high_availability_enabled"),
			"Partial Kubernetes high availability upgrade is not supported",
			"High availability upgrades enable both the control plane and load balancer. Set both high_availability_enabled values to true to upgrade the cluster.",
		),
	}
}

func expectedKubernetesClusterLoadBalancerHAPartialUpgradeDiagnostics() diag.Diagnostics {
	return diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(
			path.Root("load_balancer").AtName("high_availability_enabled"),
			"Partial Kubernetes high availability upgrade is not supported",
			"High availability upgrades enable both the control plane and load balancer. Set both high_availability_enabled values to true to upgrade the cluster.",
		),
	}
}
