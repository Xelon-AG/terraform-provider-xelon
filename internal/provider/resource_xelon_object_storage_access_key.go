package provider

import (
	"context"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*objectStorageAccessKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*objectStorageAccessKeyResource)(nil)
	_ resource.ResourceWithImportState = (*objectStorageAccessKeyResource)(nil)
)

// objectStorageAccessKeyResource is the object storage access key resource implementation.
type objectStorageAccessKeyResource struct {
	client *xelon.Client
}

// objectStorageUserResourceModel maps the object storage access key resource schema data.
type objectStorageAccessKeyResourceModel struct {
	AccessKeyID         types.String `tfsdk:"access_key_id"`
	CreatedAt           types.String `tfsdk:"created_at"`
	ID                  types.String `tfsdk:"id"`
	SecretAccessKey     types.String `tfsdk:"secret_access_key"`
	ObjectStorageUserID types.String `tfsdk:"user_id"`
}

func NewObjectStorageAccessKeyResource() resource.Resource {
	return &objectStorageAccessKeyResource{}
}

func (r *objectStorageAccessKeyResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_object_storage_access_key"
}

func (r *objectStorageAccessKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The object storage access key resource allows you to manage Xelon Object Storage Access Key for a user.

An access key represents an S3-compatible credential pair consisting of an access key ID and a secret key.
These credentials are used to authenticate applications and tools against Xelon Object Storage.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"access_key_id": schema.StringAttribute{
				MarkdownDescription: "S3 access key ID used to authenticate requests.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Timestamp when the access key was created (RFC3339).",
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the object storage access key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_access_key": schema.StringAttribute{
				MarkdownDescription: "S3 secret access key used to sign requests. Only returned at creation time and cannot be retrieved afterwards.",
				Computed:            true,
				Sensitive:           true,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "ID of the object storage user owning this access key.",
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

func (r *objectStorageAccessKeyResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *objectStorageAccessKeyResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data objectStorageAccessKeyResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	userID := data.ObjectStorageUserID.ValueString()

	tflog.Debug(ctx, "creating object storage access key", map[string]any{"user_id": userID})

	tflog.Trace(ctx, "creating object storage user token via API", map[string]any{"user_id": userID})
	token, _, err := r.client.ObjectStorages.CreateUserToken(ctx, userID)
	if err != nil {
		response.Diagnostics.AddError("Unable to create object storage access key", err.Error())
		return
	}

	tflog.Debug(ctx, "created object storage access key", map[string]any{
		"user_id":       userID,
		"id":            token.ID,
		"access_key_id": token.AccessKey,
	})

	// map API response to Terraform state
	data.fromAPI(token, userID)
	// secret is only available at creation time
	data.SecretAccessKey = types.StringValue(token.SecretKey)
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *objectStorageAccessKeyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data objectStorageAccessKeyResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	userID := data.ObjectStorageUserID.ValueString()
	tokenID := data.ID.ValueString()

	tflog.Debug(ctx, "reading object storage access key", map[string]any{
		"user_id": userID,
		"id":      tokenID,
	})

	tflog.Trace(ctx, "fetching object storage user via API (token lookup in list)", map[string]any{"user_id": userID})
	user, resp, err := r.client.ObjectStorages.GetUser(ctx, userID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// user not found, remove from state
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get object storage user", err.Error())
		return
	}

	found := false
	for _, token := range user.Tokens {
		if token.ID == tokenID {
			data.fromAPI(&token, userID)
			found = true
			break
		}
	}

	if !found {
		// access key not found, remove from state
		response.State.RemoveResource(ctx)
		return
	}

	// map API response to Terraform state
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *objectStorageAccessKeyResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	// no-op: update is not supported
}

func (r *objectStorageAccessKeyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data objectStorageAccessKeyResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	userID := data.ObjectStorageUserID.ValueString()
	tokenID := data.ID.ValueString()

	tflog.Debug(ctx, "deleting object storage access key", map[string]any{
		"user_id": userID,
		"id":      tokenID,
	})

	tflog.Trace(ctx, "deleting object storage user token via API", map[string]any{
		"user_id":  userID,
		"token_id": tokenID,
	})
	_, err := r.client.ObjectStorages.DeleteUserToken(ctx, userID, tokenID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete object storage access key", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted object storage access key", map[string]any{
		"user_id": userID,
		"id":      tokenID,
	})
}

func (r *objectStorageAccessKeyResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	parts := strings.SplitN(request.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		response.Diagnostics.AddError("Invalid import identifier", "Expected format: <user_id>/<id>")
		return
	}

	userID := parts[0]
	tokenID := parts[1]

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), tokenID)...)
}

func (m *objectStorageAccessKeyResourceModel) fromAPI(objectStorageUserToken *xelon.ObjectStorageUserToken, objectStorageUserID string) {
	m.AccessKeyID = types.StringValue(objectStorageUserToken.AccessKey)
	m.CreatedAt = helper.FormatTimeRFC3339(objectStorageUserToken.CreatedAt)
	m.ID = types.StringValue(objectStorageUserToken.ID)
	m.ObjectStorageUserID = types.StringValue(objectStorageUserID)
}
