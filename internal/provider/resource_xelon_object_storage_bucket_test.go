package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestResourceXelonObjectStorageBucket_Schema_ObjectLockRetentionDays(t *testing.T) {
	testCases := map[string]struct {
		value          types.Int64
		expectHasError bool
	}{
		"minimum is valid": {
			value: types.Int64Value(1),
		},
		"zero is invalid": {
			value:          types.Int64Value(0),
			expectHasError: true,
		},
		"maximum is valid": {
			value: types.Int64Value(objectStorageBucketObjectLockRetentionDaysMaximum),
		},
		"greater than maximum is invalid": {
			value:          types.Int64Value(objectStorageBucketObjectLockRetentionDaysMaximum + 1),
			expectHasError: true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			response := testObjectStorageBucketInt64ValidatorResponse(t, "object_lock_retention_days", testCase.value)
			assert.Equal(t, testCase.expectHasError, response.Diagnostics.HasError())
		})
	}
}

func TestResourceXelonObjectStorageBucket_ValidateConfig(t *testing.T) {
	testCases := map[string]struct {
		model           objectStorageBucketResourceModel
		expectHasError  bool
		expectedSummary string
		expectedPath    path.Path
	}{
		"object lock enabled with versioning disabled is rejected": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Value(30),
				types.BoolValue(false),
			),
			expectHasError:  true,
			expectedSummary: "Object Lock requires versioning",
			expectedPath:    path.Root("versioning_enabled"),
		},
		"retention configured with object lock disabled is rejected": {
			model: testObjectStorageBucketModel(
				types.BoolValue(false),
				types.Int64Value(30),
				types.BoolValue(true),
			),
			expectHasError:  true,
			expectedSummary: "Object Lock retention requires Object Lock",
			expectedPath:    path.Root("object_lock_retention_days"),
		},
		"retention configured with object lock omitted is rejected": {
			model: testObjectStorageBucketModel(
				types.BoolNull(),
				types.Int64Value(30),
				types.BoolValue(true),
			),
			expectHasError:  true,
			expectedSummary: "Object Lock retention requires Object Lock",
			expectedPath:    path.Root("object_lock_retention_days"),
		},
		"object lock enabled with versioning and no retention is deferred to plan validation": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Null(),
				types.BoolValue(true),
			),
		},
		"object lock enabled with versioning and valid retention is valid": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Value(30),
				types.BoolValue(true),
			),
		},
		"object lock disabled with no retention is valid": {
			model: testObjectStorageBucketModel(
				types.BoolValue(false),
				types.Int64Null(),
				types.BoolValue(false),
			),
		},
		"unknown object lock does not emit speculative diagnostics": {
			model: testObjectStorageBucketModel(
				types.BoolUnknown(),
				types.Int64Value(30),
				types.BoolValue(false),
			),
		},
		"unknown versioning does not emit speculative diagnostics": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Null(),
				types.BoolUnknown(),
			),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			response := testObjectStorageBucketValidateConfigResponse(t, testCase.model)
			assert.Equal(t, testCase.expectHasError, response.Diagnostics.HasError())
			if testCase.expectHasError {
				require.Len(t, response.Diagnostics, 1)
				assert.Equal(t, testCase.expectedSummary, response.Diagnostics[0].Summary())
				diagnosticWithPath, ok := response.Diagnostics[0].(diag.DiagnosticWithPath)
				require.True(t, ok)
				assert.Equal(t, testCase.expectedPath.String(), diagnosticWithPath.Path().String())
			}
		})
	}
}

func TestResourceXelonObjectStorageBucket_ModifyPlan_ObjectLockRetention(t *testing.T) {
	historicalState := testObjectStorageBucketModel(
		types.BoolValue(true),
		types.Int64Null(),
		types.BoolValue(true),
	)
	historicalNameChangedPlan := historicalState
	historicalNameChangedPlan.Name = types.StringValue("replacement-bucket")
	historicalUserChangedPlan := historicalState
	historicalUserChangedPlan.ObjectStorageUserID = types.StringValue("replacement-user-id")
	historicalUnknownNamePlan := historicalState
	historicalUnknownNamePlan.Name = types.StringUnknown()
	historicalUnknownUserPlan := historicalState
	historicalUnknownUserPlan.ObjectStorageUserID = types.StringUnknown()

	retainedState := historicalState
	retainedState.ObjectLockRetentionDays = types.Int64Value(30)
	unlockedState := historicalState
	unlockedState.ObjectLockEnabled = types.BoolValue(false)

	testCases := map[string]struct {
		plan           objectStorageBucketResourceModel
		state          objectStorageBucketResourceModel
		planIsNull     bool
		stateIsNull    bool
		expectHasError bool
	}{
		"new locked bucket without retention is rejected": {
			plan:           historicalState,
			stateIsNull:    true,
			expectHasError: true,
		},
		"new locked bucket with retention is allowed": {
			plan:        retainedState,
			stateIsNull: true,
		},
		"unchanged historical bucket is allowed": {
			plan:  historicalState,
			state: historicalState,
		},
		"historical bucket name change is rejected": {
			plan:           historicalNameChangedPlan,
			state:          historicalState,
			expectHasError: true,
		},
		"historical bucket user change is rejected": {
			plan:           historicalUserChangedPlan,
			state:          historicalState,
			expectHasError: true,
		},
		"retention removal is rejected": {
			plan:           historicalState,
			state:          retainedState,
			expectHasError: true,
		},
		"enabling Object Lock without retention is rejected": {
			plan:           historicalState,
			state:          unlockedState,
			expectHasError: true,
		},
		"unknown retention is deferred": {
			plan: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Unknown(),
				types.BoolValue(true),
			),
			stateIsNull: true,
		},
		"unknown Object Lock is deferred": {
			plan: testObjectStorageBucketModel(
				types.BoolUnknown(),
				types.Int64Null(),
				types.BoolValue(true),
			),
			stateIsNull: true,
		},
		"unknown historical bucket name is deferred": {
			plan:  historicalUnknownNamePlan,
			state: historicalState,
		},
		"unknown historical bucket user is deferred": {
			plan:  historicalUnknownUserPlan,
			state: historicalState,
		},
		"destroy plan is allowed": {
			planIsNull: true,
			state:      historicalState,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			response := testObjectStorageBucketModifyPlanResponse(t, testCase.plan, testCase.state, testCase.planIsNull, testCase.stateIsNull)

			assert.Equal(t, testCase.expectHasError, response.Diagnostics.HasError())
			if testCase.expectHasError {
				require.Len(t, response.Diagnostics, 1)
				assert.Equal(t, "Object Lock requires retention", response.Diagnostics[0].Summary())
				diagnosticWithPath, ok := response.Diagnostics[0].(diag.DiagnosticWithPath)
				require.True(t, ok)
				assert.Equal(t, path.Root("object_lock_retention_days").String(), diagnosticWithPath.Path().String())
			}
		})
	}
}

func TestResourceXelonObjectStorageBucket_Create_ObjectLockInvariant(t *testing.T) {
	testCases := map[string]struct {
		model           objectStorageBucketResourceModel
		expectedSummary string
		expectedPath    path.Path
	}{
		"missing retention is rejected": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Null(),
				types.BoolValue(true),
			),
			expectedSummary: "Object Lock requires retention",
			expectedPath:    path.Root("object_lock_retention_days"),
		},
		"unknown retention is rejected": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Unknown(),
				types.BoolValue(true),
			),
			expectedSummary: "Object Lock requires retention",
			expectedPath:    path.Root("object_lock_retention_days"),
		},
		"disabled versioning is rejected": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Value(30),
				types.BoolValue(false),
			),
			expectedSummary: "Object Lock requires versioning",
			expectedPath:    path.Root("versioning_enabled"),
		},
		"retention below minimum is rejected": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Value(objectStorageBucketObjectLockRetentionDaysMinimum-1),
				types.BoolValue(true),
			),
			expectedSummary: "Invalid Object Lock retention",
			expectedPath:    path.Root("object_lock_retention_days"),
		},
		"retention above maximum is rejected": {
			model: testObjectStorageBucketModel(
				types.BoolValue(true),
				types.Int64Value(objectStorageBucketObjectLockRetentionDaysMaximum+1),
				types.BoolValue(true),
			),
			expectedSummary: "Invalid Object Lock retention",
			expectedPath:    path.Root("object_lock_retention_days"),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			response := testObjectStorageBucketCreateResponse(t, testCase.model)

			require.True(t, response.Diagnostics.HasError())
			require.Len(t, response.Diagnostics, 1)
			assert.Equal(t, testCase.expectedSummary, response.Diagnostics[0].Summary())
			diagnosticWithPath, ok := response.Diagnostics[0].(diag.DiagnosticWithPath)
			require.True(t, ok)
			assert.Equal(t, testCase.expectedPath.String(), diagnosticWithPath.Path().String())
		})
	}
}

func TestResourceXelonObjectStorageBucket_Model_FromAPI(t *testing.T) {
	ctx := context.Background()
	bucket := &xelon.ObjectStorageBucket{
		CreatedAt:                mustTime(t, "2025-10-27T13:19:56Z"),
		ID:                       "zone1.4711.1",
		Name:                     "test-bucket",
		ObjectLockEnabled:        true,
		ObjectLockRetentionDays:  90,
		ObjectStorageUserID:      "api-user-id",
		RegionName:               "Swiss",
		RegionReplicationEnabled: true,
		S3Endpoints:              []string{"https://ch1-s3.xelon.io"},
		Tenant:                   &xelon.Tenant{ID: "tenant-id"},
		VersioningEnabled:        true,
	}

	expectedEndpoints, diags := types.SetValueFrom(
		ctx,
		types.StringType,
		[]string{"https://ch1-s3.xelon.io"},
	)
	require.False(t, diags.HasError())

	expected := objectStorageBucketResourceModel{
		CreatedAt:                types.StringValue("2025-10-27T13:19:56Z"),
		ID:                       types.StringValue("zone1.4711.1"),
		Name:                     types.StringValue("test-bucket"),
		ObjectLockEnabled:        types.BoolValue(true),
		ObjectLockRetentionDays:  types.Int64Value(90),
		ObjectStorageUserID:      types.StringValue("terraform-user-id"),
		RegionReplicationEnabled: types.BoolValue(true),
		S3Endpoints:              expectedEndpoints,
		TenantID:                 types.StringValue("tenant-id"),
		VersioningEnabled:        types.BoolValue(true),
	}

	var actual objectStorageBucketResourceModel
	diags = actual.fromAPI(ctx, bucket, "terraform-user-id")

	require.False(t, diags.HasError())
	assert.Equal(t, expected, actual)
}

func TestResourceXelonObjectStorageBucket_Model_FromAPI_ZeroRetentionMapsToNull(t *testing.T) {
	testCases := map[string]struct {
		objectLockEnabled bool
	}{
		"unlocked bucket": {
			objectLockEnabled: false,
		},
		"locked bucket without default retention": {
			objectLockEnabled: true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			var model objectStorageBucketResourceModel
			diags := model.fromAPI(context.Background(), &xelon.ObjectStorageBucket{
				ID:                      "zone1.4711.1",
				Name:                    "test-bucket",
				ObjectLockEnabled:       testCase.objectLockEnabled,
				ObjectLockRetentionDays: 0,
			}, "user-id")

			require.False(t, diags.HasError())
			assert.Equal(t, types.BoolValue(testCase.objectLockEnabled), model.ObjectLockEnabled)
			assert.True(t, model.ObjectLockRetentionDays.IsNull())
		})
	}
}

func TestResourceXelonObjectStorageBucket_Model_FromAPI_NilOptionalFields(t *testing.T) {
	model := objectStorageBucketResourceModel{
		TenantID: types.StringValue("previous-tenant-id"),
	}
	bucket := &xelon.ObjectStorageBucket{
		ID:   "zone1.4711.1",
		Name: "test-bucket",
	}

	diags := model.fromAPI(context.Background(), bucket, "user-id")

	require.False(t, diags.HasError())
	assert.True(t, model.CreatedAt.IsNull())
	assert.Equal(t, types.BoolValue(false), model.ObjectLockEnabled)
	assert.True(t, model.ObjectLockRetentionDays.IsNull())
	assert.True(t, model.S3Endpoints.IsNull())
	assert.True(t, model.TenantID.IsNull())
}

func TestResourceXelonObjectStorageBucket_ImportID_Parse(t *testing.T) {
	userID, bucketName, err := parseObjectStorageBucketImportID("user-123/test-bucket")

	require.NoError(t, err)
	assert.Equal(t, "user-123", userID)
	assert.Equal(t, "test-bucket", bucketName)
}

func TestResourceXelonObjectStorageBucket_ImportID_ParseInvalid(t *testing.T) {
	testCases := []string{
		"",
		"user-123",
		"/test-bucket",
		"user-123/",
	}

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			_, _, err := parseObjectStorageBucketImportID(testCase)
			require.Error(t, err)
		})
	}
}

func TestResourceXelonObjectStorageBucket_ImportState_InvalidIDDiagnostic(t *testing.T) {
	response := &resource.ImportStateResponse{}

	NewObjectStorageBucketResource().(*objectStorageBucketResource).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "user-123",
	}, response)

	require.True(t, response.Diagnostics.HasError())
	require.Len(t, response.Diagnostics, 1)
	assert.Equal(t, "Invalid import identifier", response.Diagnostics[0].Summary())
	assert.Equal(t, "Expected format: <user-id>/<bucket-name>", response.Diagnostics[0].Detail())
}

func testObjectStorageBucketResourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	r := NewObjectStorageBucketResource()
	response := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, response)
	require.False(t, response.Diagnostics.HasError())

	return response.Schema
}

func testObjectStorageBucketInt64ValidatorResponse(t *testing.T, attributeName string, value types.Int64) *validator.Int64Response {
	t.Helper()

	bucketSchema := testObjectStorageBucketResourceSchema(t)
	attribute, ok := bucketSchema.Attributes[attributeName].(schema.Int64Attribute)
	require.True(t, ok)
	require.Len(t, attribute.Validators, 1)

	response := &validator.Int64Response{}
	attribute.Validators[0].ValidateInt64(context.Background(), validator.Int64Request{
		Path:        path.Root(attributeName),
		ConfigValue: value,
	}, response)

	return response
}

func testObjectStorageBucketValidateConfigResponse(t *testing.T, model objectStorageBucketResourceModel) *resource.ValidateConfigResponse {
	t.Helper()

	ctx := context.Background()
	bucketSchema := testObjectStorageBucketResourceSchema(t)
	plan := testObjectStorageBucketResourcePlan(t, ctx, bucketSchema, model)
	config := tfsdk.Config{
		Schema: bucketSchema,
		Raw:    plan.Raw,
	}

	response := &resource.ValidateConfigResponse{}
	NewObjectStorageBucketResource().(*objectStorageBucketResource).ValidateConfig(ctx, resource.ValidateConfigRequest{
		Config: config,
	}, response)

	return response
}

func testObjectStorageBucketModifyPlanResponse(
	t *testing.T,
	planModel objectStorageBucketResourceModel,
	stateModel objectStorageBucketResourceModel,
	planIsNull bool,
	stateIsNull bool,
) *resource.ModifyPlanResponse {
	t.Helper()

	ctx := context.Background()
	bucketSchema := testObjectStorageBucketResourceSchema(t)
	terraformType := bucketSchema.Type().TerraformType(ctx)

	plan := tfsdk.Plan{
		Schema: bucketSchema,
		Raw:    tftypes.NewValue(terraformType, nil),
	}
	if !planIsNull {
		plan = testObjectStorageBucketResourcePlan(t, ctx, bucketSchema, planModel)
	}

	state := tfsdk.State{
		Schema: bucketSchema,
		Raw:    tftypes.NewValue(terraformType, nil),
	}
	if !stateIsNull {
		statePlan := testObjectStorageBucketResourcePlan(t, ctx, bucketSchema, stateModel)
		state.Raw = statePlan.Raw
	}

	response := &resource.ModifyPlanResponse{}
	NewObjectStorageBucketResource().(*objectStorageBucketResource).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:  plan,
		State: state,
	}, response)

	return response
}

func testObjectStorageBucketCreateResponse(t *testing.T, model objectStorageBucketResourceModel) *resource.CreateResponse {
	t.Helper()

	ctx := context.Background()
	bucketSchema := testObjectStorageBucketResourceSchema(t)
	plan := testObjectStorageBucketResourcePlan(t, ctx, bucketSchema, model)

	response := &resource.CreateResponse{}
	NewObjectStorageBucketResource().(*objectStorageBucketResource).Create(ctx, resource.CreateRequest{
		Plan: plan,
	}, response)

	return response
}

func testObjectStorageBucketResourcePlan(t *testing.T, ctx context.Context, bucketSchema schema.Schema, model objectStorageBucketResourceModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: bucketSchema}
	diags := plan.Set(ctx, &model)
	require.False(t, diags.HasError())

	return plan
}

func testObjectStorageBucketModel(objectLockEnabled types.Bool, objectLockRetentionDays types.Int64, versioningEnabled types.Bool) objectStorageBucketResourceModel {
	return objectStorageBucketResourceModel{
		CreatedAt:                types.StringValue("2025-10-27T13:19:56Z"),
		ID:                       types.StringValue("zone1.4711.1"),
		Name:                     types.StringValue("test-bucket"),
		ObjectLockEnabled:        objectLockEnabled,
		ObjectLockRetentionDays:  objectLockRetentionDays,
		ObjectStorageUserID:      types.StringValue("user-id"),
		RegionReplicationEnabled: types.BoolValue(true),
		S3Endpoints:              types.SetNull(types.StringType),
		TenantID:                 types.StringValue("tenant-id"),
		VersioningEnabled:        versioningEnabled,
	}
}
