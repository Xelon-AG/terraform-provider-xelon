package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const tenantUserEmailTakenValidationMessage = "The email has already been taken."

var (
	_ resource.Resource                = (*tenantUserResource)(nil)
	_ resource.ResourceWithConfigure   = (*tenantUserResource)(nil)
	_ resource.ResourceWithImportState = (*tenantUserResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*tenantUserResource)(nil)
)

// tenantUserResource is the tenant user resource implementation.
type tenantUserResource struct {
	client *xelon.Client
}

// tenantUserResourceModel maps the tenant user resource schema data.
type tenantUserResourceModel struct {
	Email                 types.String `tfsdk:"email"`
	FirstName             types.String `tfsdk:"first_name"`
	ID                    types.String `tfsdk:"id"`
	LastName              types.String `tfsdk:"last_name"`
	Password              types.String `tfsdk:"password"`
	Permissions           types.Set    `tfsdk:"permissions"` // []types.String
	RequirePasswordChange types.Bool   `tfsdk:"require_password_change"`
	Roles                 types.Set    `tfsdk:"roles"` // []types.String
	SendWelcomeEmail      types.Bool   `tfsdk:"send_welcome_email"`
	TenantID              types.String `tfsdk:"tenant_id"`
}

func NewTenantUserResource() resource.Resource {
	return &tenantUserResource{}
}

func (r *tenantUserResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_tenant_user"
}

func (r *tenantUserResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The tenant user resource allows you to manage a user that belongs to a Xelon tenant.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address of the tenant user. Changing this value requires replacing the user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the tenant user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"first_name": schema.StringAttribute{
				MarkdownDescription: "First name of the tenant user.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"last_name": schema.StringAttribute{
				MarkdownDescription: "Last name of the tenant user.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for the tenant user.",
				Required:            true,
				Sensitive:           true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"permissions": schema.SetAttribute{
				MarkdownDescription: "Permission API names assigned to the tenant user.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"require_password_change": schema.BoolAttribute{
				MarkdownDescription: "Whether the user must change their password after the initial login. " +
					"This value is only applied when the user is created and cannot be changed later.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"roles": schema.SetAttribute{
				MarkdownDescription: "Role API names assigned to the tenant user.",
				Required:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"send_welcome_email": schema.BoolAttribute{
				MarkdownDescription: "Whether Xelon should send the user a welcome email when the user is created. " +
					"This value is only applied when the user is created and cannot be changed later.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the tenant that owns the user. Changing this value requires replacing the user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *tenantUserResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*xelon.Client)
	if !ok {
		response.Diagnostics.AddError(
			"Unconfigured Xelon client",
			"Please report this issue to the provider developers.",
		)
		return
	}

	r.client = client
}

func (r *tenantUserResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data tenantUserResourceModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tenantID := data.TenantID.ValueString()
	email := data.Email.ValueString()
	roles, d := helper.SortedStringSetElements(ctx, data.Roles)
	response.Diagnostics.Append(d...)
	permissions, d := helper.SortedStringSetElements(ctx, data.Permissions)
	response.Diagnostics.Append(d...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &xelon.TenantUserCreateRequest{
		Email:                 email,
		FirstName:             data.FirstName.ValueString(),
		LastName:              data.LastName.ValueString(),
		Password:              data.Password.ValueString(),
		PasswordConfirmation:  data.Password.ValueString(),
		Permissions:           permissions,
		RequirePasswordChange: data.RequirePasswordChange.ValueBool(),
		Roles:                 roles,
		SendWelcomeEmail:      data.SendWelcomeEmail.ValueBool(),
	}

	tflog.Debug(ctx, "creating tenant user", map[string]any{
		"tenant_id": tenantID,
		"email":     email,
	})

	tflog.Trace(ctx, "creating tenant user via API", map[string]any{
		"tenant_id": tenantID,
		"email":     email,
	})
	createdUser, _, err := r.client.TenantUsers.Create(ctx, tenantID, createRequest)
	if err != nil {
		conflictDiags, handled := r.createTenantUserEmailConflictDiagnostics(ctx, tenantID, email, err)
		response.Diagnostics.Append(conflictDiags...)
		if handled {
			return
		}
		response.Diagnostics.AddError("Unable to create tenant user", err.Error())
		return
	}

	tflog.Debug(ctx, "created tenant user", map[string]any{
		"tenant_id": tenantID,
		"user_id":   createdUser.ID,
	})

	tflog.Debug(ctx, "refreshing tenant user state after create", map[string]any{
		"tenant_id": tenantID,
		"user_id":   createdUser.ID,
	})

	tflog.Trace(ctx, "reading tenant user via API (created user refresh)", map[string]any{
		"tenant_id": tenantID,
		"user_id":   createdUser.ID,
	})
	user, _, err := r.client.TenantUsers.Get(ctx, tenantID, createdUser.ID)
	if err != nil {
		response.Diagnostics.AddError("Unable to read tenant user", err.Error())
		return
	}
	if !user.IsActive {
		response.Diagnostics.AddError(
			"Tenant user inactive after creation",
			"The tenant user was created but is inactive when refreshed from the API.",
		)
		return
	}

	response.Diagnostics.Append(data.fromAPI(ctx, user, tenantID)...)
	if response.Diagnostics.HasError() {
		return
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *tenantUserResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data tenantUserResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tenantID := data.TenantID.ValueString()
	userID := data.ID.ValueString()

	tflog.Debug(ctx, "reading tenant user", map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	})

	tflog.Trace(ctx, "reading tenant user via API", map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	})
	user, resp, err := r.client.TenantUsers.Get(ctx, tenantID, userID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "tenant user not found; removing from state", map[string]any{
				"tenant_id": tenantID,
				"user_id":   userID,
			})
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read tenant user", err.Error())
		return
	}
	if !user.IsActive {
		tflog.Debug(ctx, "tenant user inactive; removing from state", map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
		})
		response.State.RemoveResource(ctx)
		return
	}

	response.Diagnostics.Append(data.fromAPI(ctx, user, tenantID)...)
	if response.Diagnostics.HasError() {
		return
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *tenantUserResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan tenantUserResourceModel
	var state tenantUserResourceModel

	diags := request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	diags = request.State.Get(ctx, &state)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tenantID := plan.TenantID.ValueString()
	userID := state.ID.ValueString()
	profileChanged := !plan.FirstName.Equal(state.FirstName) || !plan.LastName.Equal(state.LastName)
	passwordChanged := !plan.Password.Equal(state.Password)
	accessChanged := !plan.Roles.Equal(state.Roles) || !plan.Permissions.Equal(state.Permissions)

	tflog.Debug(ctx, "updating tenant user", map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	})

	if profileChanged {
		updateRequest := &xelon.TenantUserUpdateRequest{
			FirstName: plan.FirstName.ValueString(),
			LastName:  plan.LastName.ValueString(),
		}

		tflog.Trace(ctx, "updating tenant user profile via API", map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
		})
		_, _, err := r.client.TenantUsers.Update(ctx, tenantID, userID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update tenant user", err.Error())
			return
		}

		tflog.Debug(ctx, "updated tenant user profile", map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
		})
	}

	if passwordChanged {
		updateRequest := &xelon.TenantUserPasswordUpdateRequest{
			Password:             plan.Password.ValueString(),
			PasswordConfirmation: plan.Password.ValueString(),
		}

		tflog.Trace(ctx, "updating tenant user password via API", map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
		})
		_, err := r.client.TenantUsers.UpdatePassword(ctx, tenantID, userID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update tenant user password", err.Error())
			return
		}

		tflog.Debug(ctx, "updated tenant user password", map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
		})
	}

	if accessChanged {
		roles, d := helper.SortedStringSetElements(ctx, plan.Roles)
		response.Diagnostics.Append(d...)
		permissions, d := helper.SortedStringSetElements(ctx, plan.Permissions)
		response.Diagnostics.Append(d...)
		if response.Diagnostics.HasError() {
			return
		}

		updateRequest := &xelon.TenantUserPermissionsUpdateRequest{
			Permissions: permissions,
			Roles:       roles,
		}

		tflog.Trace(ctx, "updating tenant user roles and permissions via API", map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
			"payload":   updateRequest,
		})
		_, err := r.client.TenantUsers.UpdatePermissions(ctx, tenantID, userID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update tenant user roles and permissions", err.Error())
			return
		}

		tflog.Debug(ctx, "updated tenant user roles and permissions", map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
		})
	}

	tflog.Debug(ctx, "refreshing tenant user state after update", map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	})

	tflog.Trace(ctx, "reading tenant user via API (updated user refresh)", map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	})
	user, _, err := r.client.TenantUsers.Get(ctx, tenantID, userID)
	if err != nil {
		response.Diagnostics.AddError("Unable to read tenant user", err.Error())
		return
	}
	if !user.IsActive {
		tflog.Debug(ctx, "tenant user inactive; removing from state", map[string]any{
			"tenant_id": tenantID,
			"user_id":   userID,
		})
		response.State.RemoveResource(ctx)
		return
	}

	response.Diagnostics.Append(plan.fromAPI(ctx, user, tenantID)...)
	if response.Diagnostics.HasError() {
		return
	}

	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}

func (r *tenantUserResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data tenantUserResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tenantID := data.TenantID.ValueString()
	userID := data.ID.ValueString()

	tflog.Debug(ctx, "deleting tenant user", map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	})

	tflog.Trace(ctx, "deleting tenant user via API", map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	})
	resp, err := r.client.TenantUsers.Delete(ctx, tenantID, userID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "tenant user already absent", map[string]any{
				"tenant_id": tenantID,
				"user_id":   userID,
			})
			return
		}
		response.Diagnostics.AddError("Unable to delete tenant user", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted tenant user", map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	})
}

func (r *tenantUserResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	tenantID, userID, err := parseTenantUserImportID(request.ID)
	if err != nil {
		response.Diagnostics.AddError("Invalid import identifier", "Expected format: <tenant-id>/<user-id>")
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("tenant_id"), tenantID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), userID)...)
}

func (r *tenantUserResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	if request.Plan.Raw.IsNull() || request.State.Raw.IsNull() {
		return
	}

	// These are one-shot create instructions, not durable remote state.
	// Block post-create changes instead of forcing destructive replacement.
	var plan tenantUserResourceModel
	var state tenantUserResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	if !plan.RequirePasswordChange.IsNull() && !plan.RequirePasswordChange.IsUnknown() {
		if state.RequirePasswordChange.IsNull() || state.RequirePasswordChange.IsUnknown() {
			if plan.RequirePasswordChange.ValueBool() {
				response.Diagnostics.AddAttributeError(
					path.Root("require_password_change"),
					"Cannot change require_password_change after creation",
					`Attribute "require_password_change" cannot be changed for an existing tenant user because it is only applied during creation.`,
				)
			}
		} else if !plan.RequirePasswordChange.Equal(state.RequirePasswordChange) {
			response.Diagnostics.AddAttributeError(
				path.Root("require_password_change"),
				"Cannot change require_password_change after creation",
				`Attribute "require_password_change" cannot be changed for an existing tenant user because it is only applied during creation.`,
			)
		}
	}

	if !plan.SendWelcomeEmail.IsNull() && !plan.SendWelcomeEmail.IsUnknown() {
		if state.SendWelcomeEmail.IsNull() || state.SendWelcomeEmail.IsUnknown() {
			if plan.SendWelcomeEmail.ValueBool() {
				response.Diagnostics.AddAttributeError(
					path.Root("send_welcome_email"),
					"Cannot change send_welcome_email after creation",
					`Attribute "send_welcome_email" cannot be changed for an existing tenant user because the welcome email can only be sent during creation.`,
				)
			}
		} else if !plan.SendWelcomeEmail.Equal(state.SendWelcomeEmail) {
			response.Diagnostics.AddAttributeError(
				path.Root("send_welcome_email"),
				"Cannot change send_welcome_email after creation",
				`Attribute "send_welcome_email" cannot be changed for an existing tenant user because the welcome email can only be sent during creation.`,
			)
		}
	}
}

func (m *tenantUserResourceModel) fromAPI(ctx context.Context, user *xelon.TenantUserWithDetails, tenantID string) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Email = types.StringValue(user.Email)
	m.FirstName = types.StringValue(user.FirstName)
	m.ID = types.StringValue(user.ID)
	m.LastName = types.StringValue(user.LastName)
	m.TenantID = types.StringValue(tenantID)

	roleNames := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleNames = append(roleNames, role.Name)
	}
	roles, d := types.SetValueFrom(ctx, types.StringType, roleNames)
	diags.Append(d...)
	m.Roles = roles

	permissionNames := make([]string, 0, len(user.Permissions))
	for _, permission := range user.Permissions {
		permissionNames = append(permissionNames, permission.Name)
	}
	permissions, d := types.SetValueFrom(ctx, types.StringType, permissionNames)
	diags.Append(d...)
	m.Permissions = permissions

	return diags
}

func (r *tenantUserResource) createTenantUserEmailConflictDiagnostics(ctx context.Context, tenantID, email string, createErr error) (diag.Diagnostics, bool) {
	var diags diag.Diagnostics

	if !isTenantUserEmailTakenError(createErr) {
		return diags, false
	}

	tflog.Debug(ctx, "enriching tenant user email-conflict diagnostic", map[string]any{
		"tenant_id": tenantID,
		"email":     email,
	})

	user, found, lookupErr := r.findTenantUserByEmail(ctx, tenantID, email)
	if lookupErr != nil {
		tflog.Debug(ctx, "tenant user email-conflict lookup failed", map[string]any{
			"tenant_id": tenantID,
			"error":     lookupErr.Error(),
		})

		return diags, false
	}

	if !found {
		return diags, false
	}

	if user.IsActive {
		diags.AddAttributeError(
			path.Root("email"),
			"Tenant user email is already used by an active user",
			`An active tenant user already uses this email.
Terraform cannot create a duplicate tenant user. Import the existing user first if this resource should manage it.`,
		)
		return diags, true
	}

	diags.AddAttributeError(
		path.Root("email"),
		"Tenant user email is reserved by an inactive user",
		`An inactive tenant user reserves this email.
Terraform will not restore inactive tenant users automatically, and xelon_tenant_user cannot manage the user while it remains inactive.
Restore the tenant user or otherwise free the email outside Terraform, then run terraform apply again.`,
	)

	return diags, true
}

func (r *tenantUserResource) findTenantUserByEmail(ctx context.Context, tenantID, email string) (xelon.TenantUser, bool, error) {
	tflog.Trace(ctx, "listing tenant users via API (email-conflict lookup)", map[string]any{
		"tenant_id": tenantID,
	})

	users, errf := r.client.TenantUsers.All(ctx, tenantID, nil)

	var matchedUser xelon.TenantUser
	found := false

	for user := range users {
		if user.Email != email {
			continue
		}

		matchedUser = user
		found = true
		break
	}

	if err := errf(); err != nil {
		return xelon.TenantUser{}, false, err
	}

	return matchedUser, found, nil
}

func isTenantUserEmailTakenError(err error) bool {
	errorResponse, ok := errors.AsType[*xelon.ErrorResponse](err)
	if !ok || errorResponse.Response == nil || errorResponse.Response.StatusCode != http.StatusUnprocessableEntity {
		return false
	}

	emailValidations, ok := errorResponse.ErrorElement.Validations["email"].([]any)
	if !ok {
		return false
	}
	for _, validation := range emailValidations {
		if validation == tenantUserEmailTakenValidationMessage {
			return true
		}
	}

	return false
}

func parseTenantUserImportID(importID string) (string, string, error) {
	if strings.Count(importID, "/") != 1 {
		return "", "", errors.New("invalid import identifier")
	}

	tenantID, userID, ok := strings.Cut(importID, "/")
	if !ok || tenantID == "" || userID == "" {
		return "", "", errors.New("invalid import identifier")
	}

	return tenantID, userID, nil
}
