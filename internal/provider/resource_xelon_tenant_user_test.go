package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestResourceXelonTenantUser_Schema_RolesAndPermissions(t *testing.T) {
	userSchema := testTenantUserResourceSchema(t)

	for _, attributeName := range []string{"roles", "permissions"} {
		t.Run(attributeName, func(t *testing.T) {
			attribute, ok := userSchema.Attributes[attributeName].(schema.SetAttribute)
			require.True(t, ok)

			assert.True(t, attribute.Required)
			assert.Equal(t, types.StringType, attribute.ElementType)
			assert.NotEmpty(t, attribute.Validators)
			assert.Empty(t, attribute.PlanModifiers)
		})
	}
}

func TestResourceXelonTenantUser_Schema_ReplacementAndMutableAttributes(t *testing.T) {
	userSchema := testTenantUserResourceSchema(t)

	for _, attributeName := range []string{"email", "tenant_id"} {
		t.Run(attributeName+" requires replacement", func(t *testing.T) {
			attribute, ok := userSchema.Attributes[attributeName].(schema.StringAttribute)
			require.True(t, ok)
			require.Len(t, attribute.PlanModifiers, 1)
		})
	}

	for _, attributeName := range []string{"first_name", "last_name", "password"} {
		t.Run(attributeName+" is mutable", func(t *testing.T) {
			attribute, ok := userSchema.Attributes[attributeName].(schema.StringAttribute)
			require.True(t, ok)
			assert.Empty(t, attribute.PlanModifiers)
		})
	}

	for _, attributeName := range []string{"roles", "permissions"} {
		t.Run(attributeName+" is mutable", func(t *testing.T) {
			attribute, ok := userSchema.Attributes[attributeName].(schema.SetAttribute)
			require.True(t, ok)
			assert.Empty(t, attribute.PlanModifiers)
		})
	}
}

func TestResourceXelonTenantUser_Schema_CreateTimeBooleans(t *testing.T) {
	userSchema := testTenantUserResourceSchema(t)

	for _, attributeName := range []string{"require_password_change", "send_welcome_email"} {
		t.Run(attributeName, func(t *testing.T) {
			attribute, ok := userSchema.Attributes[attributeName].(schema.BoolAttribute)
			require.True(t, ok)

			assert.True(t, attribute.Optional)
			assert.True(t, attribute.Computed)
			assert.Empty(t, attribute.PlanModifiers)

			response := &defaults.BoolResponse{}
			attribute.Default.DefaultBool(context.Background(), defaults.BoolRequest{}, response)

			require.False(t, response.Diagnostics.HasError())
			assert.Equal(t, types.BoolValue(false), response.PlanValue)
		})
	}
}

func TestResourceXelonTenantUser_Model_FromAPI_PreservesPassword(t *testing.T) {
	ctx := context.Background()
	user := &xelon.TenantUserWithDetails{
		TenantUser: xelon.TenantUser{
			Email:     "terraform-user@example.com",
			FirstName: "Terraform",
			ID:        "user-123",
			LastName:  "User",
		},
		Roles: []xelon.TenantUserRole{
			{Name: "role-b"},
			{Name: "role-a"},
		},
		Permissions: []xelon.TenantUserPermission{
			{Name: "permission-b"},
			{Name: "permission-a"},
		},
	}
	expected := tenantUserResourceModel{
		Email:                 types.StringValue("terraform-user@example.com"),
		FirstName:             types.StringValue("Terraform"),
		ID:                    types.StringValue("user-123"),
		LastName:              types.StringValue("User"),
		Password:              types.StringValue("SecurePass123!"),
		Permissions:           testTenantUserSetValue(t, "permission-b", "permission-a"),
		RequirePasswordChange: types.BoolValue(true),
		Roles:                 testTenantUserSetValue(t, "role-b", "role-a"),
		SendWelcomeEmail:      types.BoolValue(true),
		TenantID:              types.StringValue("tenant-123"),
	}

	actual := tenantUserResourceModel{
		Password:              types.StringValue("SecurePass123!"),
		RequirePasswordChange: types.BoolValue(true),
		SendWelcomeEmail:      types.BoolValue(true),
	}
	diags := actual.fromAPI(ctx, user, "tenant-123")
	require.False(t, diags.HasError())

	assert.Equal(t, expected, actual)
}

func TestResourceXelonTenantUser_EmailConflictError_DetectsEmailTaken(t *testing.T) {
	testCases := map[string]struct {
		statusCode  int
		validations map[string]any
		expected    bool
	}{
		"email taken 422 is intercepted": {
			statusCode: http.StatusUnprocessableEntity,
			validations: map[string]any{
				"email": []any{tenantUserEmailTakenValidationMessage},
			},
			expected: true,
		},
		"email 422 with different message is not intercepted": {
			statusCode: http.StatusUnprocessableEntity,
			validations: map[string]any{
				"email": []any{"Some other email validation error."},
			},
			expected: false,
		},
		"non-email 422 is not intercepted": {
			statusCode: http.StatusUnprocessableEntity,
			validations: map[string]any{
				"password": []any{tenantUserEmailTakenValidationMessage},
			},
		},
		"non-422 is not intercepted": {
			statusCode: http.StatusInternalServerError,
			validations: map[string]any{
				"email": []any{tenantUserEmailTakenValidationMessage},
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			err := testTenantUserErrorResponse(t, testCase.statusCode, testCase.validations)

			assert.Equal(t, testCase.expected, isTenantUserEmailTakenError(err))
		})
	}
}

func TestResourceXelonTenantUser_ModifyPlan_AllowsCreate(t *testing.T) {
	response := testTenantUserModifyPlanResponse(t, tenantUserModifyPlanTestCase{
		planRequirePasswordChange: types.BoolValue(true),
		planSendWelcomeEmail:      types.BoolValue(true),
		stateIsNull:               true,
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonTenantUser_ModifyPlan_AllowsDestroy(t *testing.T) {
	response := testTenantUserModifyPlanResponse(t, tenantUserModifyPlanTestCase{
		planIsNull:                 true,
		stateRequirePasswordChange: types.BoolValue(true),
		stateSendWelcomeEmail:      types.BoolValue(true),
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonTenantUser_ModifyPlan_BlocksRequirePasswordChangeAfterCreation(t *testing.T) {
	response := testTenantUserModifyPlanResponse(t, tenantUserModifyPlanTestCase{
		planRequirePasswordChange:  types.BoolValue(true),
		planSendWelcomeEmail:       types.BoolValue(false),
		stateRequirePasswordChange: types.BoolValue(false),
		stateSendWelcomeEmail:      types.BoolValue(false),
	})

	require.True(t, response.Diagnostics.HasError())
	require.Len(t, response.Diagnostics, 1)
	assert.True(t, response.Diagnostics.Equal(expectedTenantUserRequirePasswordChangeAfterCreationDiagnostics()))
}

func TestResourceXelonTenantUser_ModifyPlan_BlocksSendWelcomeEmailAfterCreation(t *testing.T) {
	response := testTenantUserModifyPlanResponse(t, tenantUserModifyPlanTestCase{
		planRequirePasswordChange:  types.BoolValue(false),
		planSendWelcomeEmail:       types.BoolValue(true),
		stateRequirePasswordChange: types.BoolValue(false),
		stateSendWelcomeEmail:      types.BoolValue(false),
	})

	require.True(t, response.Diagnostics.HasError())
	require.Len(t, response.Diagnostics, 1)
	assert.True(t, response.Diagnostics.Equal(expectedTenantUserSendWelcomeEmailAfterCreationDiagnostics()))
}

func TestResourceXelonTenantUser_ModifyPlan_ImportedNullStateNormalizesToDefaultFalse(t *testing.T) {
	response := testTenantUserModifyPlanResponse(t, tenantUserModifyPlanTestCase{
		planRequirePasswordChange:  types.BoolValue(false),
		planSendWelcomeEmail:       types.BoolValue(false),
		stateRequirePasswordChange: types.BoolNull(),
		stateSendWelcomeEmail:      types.BoolNull(),
	})

	assert.False(t, response.Diagnostics.HasError())
}

func TestResourceXelonTenantUser_ModifyPlan_ImportedNullStateCannotChangeToTrue(t *testing.T) {
	response := testTenantUserModifyPlanResponse(t, tenantUserModifyPlanTestCase{
		planRequirePasswordChange:  types.BoolValue(true),
		planSendWelcomeEmail:       types.BoolValue(true),
		stateRequirePasswordChange: types.BoolNull(),
		stateSendWelcomeEmail:      types.BoolNull(),
	})

	require.True(t, response.Diagnostics.HasError())
	require.Len(t, response.Diagnostics, 2)
	assert.True(t, response.Diagnostics.Equal(expectedTenantUserCreateTimeBooleanAfterCreationDiagnostics()))
}

func TestResourceXelonTenantUser_ImportID_Parse(t *testing.T) {
	tenantID, userID, err := parseTenantUserImportID("tenant-123/user-123")

	require.NoError(t, err)
	assert.Equal(t, "tenant-123", tenantID)
	assert.Equal(t, "user-123", userID)
}

func TestResourceXelonTenantUser_ImportID_ParseInvalid(t *testing.T) {
	testCases := []string{
		"",
		"tenant-123",
		"/user-123",
		"tenant-123/",
		"tenant-123/user-123/extra",
	}

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			_, _, err := parseTenantUserImportID(testCase)
			require.Error(t, err)
		})
	}
}

func TestResourceXelonTenantUser_ImportState_InvalidIDDiagnostic(t *testing.T) {
	response := &resource.ImportStateResponse{}

	NewTenantUserResource().(*tenantUserResource).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "tenant-123",
	}, response)

	require.True(t, response.Diagnostics.HasError())
	require.Len(t, response.Diagnostics, 1)
	assert.True(t, response.Diagnostics.Equal(expectedTenantUserInvalidImportIDDiagnostics()))
}

func testTenantUserResourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	r := NewTenantUserResource()
	response := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, response)
	require.False(t, response.Diagnostics.HasError())

	return response.Schema
}

type tenantUserModifyPlanTestCase struct {
	planIsNull                 bool
	planRequirePasswordChange  types.Bool
	planSendWelcomeEmail       types.Bool
	stateIsNull                bool
	stateRequirePasswordChange types.Bool
	stateSendWelcomeEmail      types.Bool
}

func expectedTenantUserRequirePasswordChangeAfterCreationDiagnostics() diag.Diagnostics {
	return diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(
			path.Root("require_password_change"),
			"Cannot change require_password_change after creation",
			`Attribute "require_password_change" cannot be changed for an existing tenant user because it is only applied during creation.`,
		),
	}
}

func expectedTenantUserSendWelcomeEmailAfterCreationDiagnostics() diag.Diagnostics {
	return diag.Diagnostics{
		diag.NewAttributeErrorDiagnostic(
			path.Root("send_welcome_email"),
			"Cannot change send_welcome_email after creation",
			`Attribute "send_welcome_email" cannot be changed for an existing tenant user because the welcome email can only be sent during creation.`,
		),
	}
}

func expectedTenantUserCreateTimeBooleanAfterCreationDiagnostics() diag.Diagnostics {
	diags := expectedTenantUserRequirePasswordChangeAfterCreationDiagnostics()
	diags.Append(expectedTenantUserSendWelcomeEmailAfterCreationDiagnostics()...)

	return diags
}

func expectedTenantUserInvalidImportIDDiagnostics() diag.Diagnostics {
	return diag.Diagnostics{
		diag.NewErrorDiagnostic(
			"Invalid import identifier",
			"Expected format: <tenant-id>/<user-id>",
		),
	}
}

func testTenantUserModifyPlanResponse(t *testing.T, testCase tenantUserModifyPlanTestCase) *resource.ModifyPlanResponse {
	t.Helper()

	ctx := context.Background()
	userSchema := testTenantUserResourceSchema(t)

	plan := tfsdk.Plan{
		Schema: userSchema,
		Raw:    tftypes.NewValue(userSchema.Type().TerraformType(ctx), nil),
	}
	if !testCase.planIsNull {
		plan = testTenantUserResourcePlan(t, ctx, userSchema, testTenantUserModel(t, tenantUserModelOptions{
			requirePasswordChange: testCase.planRequirePasswordChange,
			sendWelcomeEmail:      testCase.planSendWelcomeEmail,
		}))
	}

	state := tfsdk.State{
		Schema: userSchema,
		Raw:    tftypes.NewValue(userSchema.Type().TerraformType(ctx), nil),
	}
	if !testCase.stateIsNull {
		statePlan := testTenantUserResourcePlan(t, ctx, userSchema, testTenantUserModel(t, tenantUserModelOptions{
			requirePasswordChange: testCase.stateRequirePasswordChange,
			sendWelcomeEmail:      testCase.stateSendWelcomeEmail,
		}))
		state.Raw = statePlan.Raw
	}

	response := &resource.ModifyPlanResponse{}
	NewTenantUserResource().(*tenantUserResource).ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:  plan,
		State: state,
	}, response)

	return response
}

func testTenantUserResourcePlan(t *testing.T, ctx context.Context, userSchema schema.Schema, model tenantUserResourceModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: userSchema}
	diags := plan.Set(ctx, &model)
	require.False(t, diags.HasError())

	return plan
}

type tenantUserModelOptions struct {
	permissions           []string
	requirePasswordChange types.Bool
	roles                 []string
	sendWelcomeEmail      types.Bool
}

func testTenantUserModel(t *testing.T, options tenantUserModelOptions) tenantUserResourceModel {
	t.Helper()

	permissions := options.permissions
	if permissions == nil {
		permissions = []string{"permission-a"}
	}

	requirePasswordChange := options.requirePasswordChange

	roles := options.roles
	if roles == nil {
		roles = []string{"role-a"}
	}

	sendWelcomeEmail := options.sendWelcomeEmail

	return tenantUserResourceModel{
		Email:                 types.StringValue("terraform-user@example.com"),
		FirstName:             types.StringValue("Terraform"),
		ID:                    types.StringValue("user-123"),
		LastName:              types.StringValue("User"),
		Password:              types.StringValue("SecurePass123!"),
		Permissions:           testTenantUserSetValue(t, permissions...),
		RequirePasswordChange: requirePasswordChange,
		Roles:                 testTenantUserSetValue(t, roles...),
		SendWelcomeEmail:      sendWelcomeEmail,
		TenantID:              types.StringValue("tenant-123"),
	}
}

func testTenantUserSetValue(t *testing.T, values ...string) types.Set {
	t.Helper()

	value, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	require.False(t, diags.HasError())

	return value
}

func testTenantUserErrorResponse(t *testing.T, statusCode int, validations map[string]any) error {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, "https://example.test/tenants/tenant-123/users", nil)
	require.NoError(t, err)

	return &xelon.ErrorResponse{
		Response: &xelon.Response{Response: &http.Response{
			Request:    req,
			StatusCode: statusCode,
		}},
		ErrorElement: xelon.ErrorElement{
			Validations: validations,
		},
	}
}
