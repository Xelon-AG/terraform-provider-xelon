package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestResourceXelonObjectStorageUser_Model_FromAPI(t *testing.T) {
	ctx := context.Background()
	user := &xelon.ObjectStorageUser{
		ID:                       "user-123",
		Name:                     "test-user",
		QuotaGB:                  500,
		RegionReplicationEnabled: true,
		S3Endpoints:              []string{"https://ch1-s3.xelon.io"},
		Tenant:                   &xelon.Tenant{ID: "tenant-id"},
	}

	expectedEndpoints, diags := types.SetValueFrom(
		ctx,
		types.StringType,
		[]string{"https://ch1-s3.xelon.io"},
	)
	require.False(t, diags.HasError())

	expected := objectStorageUserResourceModel{
		ID:                       types.StringValue("user-123"),
		Name:                     types.StringValue("test-user"),
		Region:                   types.StringValue("zh1"),
		RegionReplicationEnabled: types.BoolValue(true),
		S3Endpoints:              expectedEndpoints,
		StorageLimit:             types.Int64Value(500),
		TenantID:                 types.StringValue("tenant-id"),
	}

	var actual objectStorageUserResourceModel
	diags = actual.fromAPI(ctx, user, "zh1")

	require.False(t, diags.HasError())
	assert.Equal(t, expected, actual)
}

func TestResourceXelonObjectStorageUser_Model_FromAPI_NilOptionalFields(t *testing.T) {
	model := objectStorageUserResourceModel{
		TenantID: types.StringValue("previous-tenant-id"),
	}
	user := &xelon.ObjectStorageUser{
		ID:      "user-123",
		Name:    "test-user",
		QuotaGB: 100,
	}

	diags := model.fromAPI(context.Background(), user, "zh1")

	require.False(t, diags.HasError())
	assert.Equal(t, types.StringValue("user-123"), model.ID)
	assert.Equal(t, types.StringValue("test-user"), model.Name)
	assert.Equal(t, types.StringValue("zh1"), model.Region)
	assert.Equal(t, types.BoolValue(false), model.RegionReplicationEnabled)
	assert.True(t, model.S3Endpoints.IsNull())
	assert.Equal(t, types.Int64Value(100), model.StorageLimit)
	assert.Equal(t, types.StringValue("previous-tenant-id"), model.TenantID)
}

func TestResourceXelonObjectStorageUser_ImportID_Parse(t *testing.T) {
	region, objectStorageUserID, err := parseObjectStorageUserImportID("zh1/user-123")

	require.NoError(t, err)
	assert.Equal(t, "zh1", region)
	assert.Equal(t, "user-123", objectStorageUserID)
}

func TestResourceXelonObjectStorageUser_ImportID_ParseInvalid(t *testing.T) {
	testCases := []string{
		"",
		"zh1",
		"/user-123",
		"zh1/",
	}

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			_, _, err := parseObjectStorageUserImportID(testCase)
			require.Error(t, err)
		})
	}
}

func TestResourceXelonObjectStorageUser_ImportState_InvalidIDDiagnostic(t *testing.T) {
	response := &resource.ImportStateResponse{}

	NewObjectStorageUserResource().(*objectStorageUserResource).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "zh1",
	}, response)

	require.True(t, response.Diagnostics.HasError())
	require.Len(t, response.Diagnostics, 1)
	assert.Equal(t, "Invalid import identifier", response.Diagnostics[0].Summary())
	assert.Equal(t, "Expected format: <region>/<id>", response.Diagnostics[0].Detail())
}
