package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource              = (*objectStorageUserResource)(nil)
	_ resource.ResourceWithConfigure = (*objectStorageUserResource)(nil)
)

// objectStorageUserResource is the object storage user resource implementation.
type objectStorageUserResource struct {
	client *xelon.Client
}

// objectStorageUserResourceModel maps the object storage user resource schema data.
type objectStorageUserResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Region                 types.String `tfsdk:"region"`
	S3Endpoints            types.Set    `tfsdk:"s3_endpoints"` // []types.String
	StorageLimit           types.Int64  `tfsdk:"storage_limit"`
	TenantID               types.String `tfsdk:"tenant_id"`
	ZoneReplicationEnabled types.Bool   `tfsdk:"zone_replication_enabled"`
}

func NewObjectStorageUserResource() resource.Resource {
	return &objectStorageUserResource{}
}

func (r *objectStorageUserResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_object_storage_user"
}

func (r *objectStorageUserResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The object storage user resource allows you to manage Xelon Object Storage users.

Users are used to create access keys and manage access to Object Storage buckets.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the object storage user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the object storage user.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "The region of the object storage user.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"s3_endpoints": schema.SetAttribute{
				MarkdownDescription: "The set of S3-compatible endpoint URLs that can be used to access object storage.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"storage_limit": schema.Int64Attribute{
				MarkdownDescription: "The storage limit of the object storage user in GB.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.OneOf(
						100, 250, 500, 1_000, 2_000, 5_000,
						10_000, 15_000, 20_000, 30_000, 40_000, 50_000,
					),
				},
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the tenant that owns the object storage user.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"zone_replication_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether zone replication is enabled for the object storage user.",
				Computed:            true,
			},
		},
	}
}

func (r *objectStorageUserResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *objectStorageUserResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data objectStorageUserResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	region := data.Region.ValueString()

	createRequest := &xelon.ObjectStorageUserCreateRequest{
		Name:     data.Name.ValueString(),
		QuotaGB:  int(data.StorageLimit.ValueInt64()),
		RegionID: region,
	}
	if !data.TenantID.IsNull() && !data.TenantID.IsUnknown() {
		createRequest.TenantID = data.TenantID.ValueString()
	}
	tflog.Debug(ctx, "Creating object storage user", map[string]any{"payload": createRequest})
	createdObjectStorageUser, _, err := r.client.ObjectStorages.CreateUser(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create object storage user", err.Error())
		return
	}
	tflog.Debug(ctx, "Created object storage user", map[string]any{"data": createdObjectStorageUser})

	objectStorageUserID := createdObjectStorageUser.ID

	tflog.Debug(ctx, "Removing API-generated default access keys for the object storage user",
		map[string]any{"object_storage_user_id": objectStorageUserID},
	)
	for _, generatedAccessKey := range createdObjectStorageUser.Tokens {
		_, err := r.client.ObjectStorages.DeleteUserToken(ctx, objectStorageUserID, generatedAccessKey.ID)
		if err != nil {
			response.Diagnostics.AddError("Unable to delete API-generated default access key", err.Error())
			return
		}
	}

	tflog.Debug(ctx, "Getting object storage user with enriched properties", map[string]any{"object_storage_user_id": objectStorageUserID})
	objectStorageUser, _, err := r.client.ObjectStorages.GetUser(ctx, objectStorageUserID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get object storage user", err.Error())
		return
	}
	tflog.Debug(ctx, "Got object storage user with enriched properties", map[string]any{"data": objectStorageUser})

	// map response body to attributes
	response.Diagnostics.Append(data.fromAPI(ctx, objectStorageUser, region)...)
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *objectStorageUserResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data objectStorageUserResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	objectStorageUserID := data.ID.ValueString()
	region := data.Region.ValueString()

	tflog.Debug(ctx, "Getting object storage user", map[string]any{"object_storage_user_id": objectStorageUserID})
	objectStorageUser, resp, err := r.client.ObjectStorages.GetUser(ctx, objectStorageUserID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the object storage user is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get object storage user", err.Error())
		return
	}
	tflog.Debug(ctx, "Got object storage user", map[string]any{"data": objectStorageUser})

	// map response body to attributes
	response.Diagnostics.Append(data.fromAPI(ctx, objectStorageUser, region)...)
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *objectStorageUserResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data objectStorageUserResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	objectStorageUserID := data.ID.ValueString()
	region := data.Region.ValueString()

	updateRequest := &xelon.ObjectStorageUserUpdateRequest{
		Name:    data.Name.ValueString(),
		QuotaGB: int(data.StorageLimit.ValueInt64()),
	}
	tflog.Debug(ctx, "Updating object storage user", map[string]any{
		"object_storage_user_id": objectStorageUserID,
		"payload":                updateRequest,
	})
	updatedObjectStorageUser, _, err := r.client.ObjectStorages.UpdateUser(ctx, objectStorageUserID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update object storage user", err.Error())
		return
	}
	tflog.Debug(ctx, "Updated object storage user", map[string]any{"data": updatedObjectStorageUser})

	tflog.Debug(ctx, "Getting object storage user with enriched properties", map[string]any{"object_storage_user_id": objectStorageUserID})
	objectStorageUser, _, err := r.client.ObjectStorages.GetUser(ctx, objectStorageUserID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get object storage user", err.Error())
		return
	}
	tflog.Debug(ctx, "Got object storage user with enriched properties", map[string]any{"data": objectStorageUser})

	// map response body to attributes
	response.Diagnostics.Append(data.fromAPI(ctx, objectStorageUser, region)...)
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *objectStorageUserResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data objectStorageUserResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	objectStorageUserID := data.ID.ValueString()

	tflog.Debug(ctx, "Deleting object storage user", map[string]any{"object_storage_user_id": objectStorageUserID})
	_, err := r.client.ObjectStorages.DeleteUser(ctx, objectStorageUserID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete object storage user", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted object storage user", map[string]any{"object_storage_user_id": objectStorageUserID})
}

func (m *objectStorageUserResourceModel) fromAPI(ctx context.Context, objectStorageUser *xelon.ObjectStorageUser, region string) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(objectStorageUser.ID)
	m.Name = types.StringValue(objectStorageUser.Name)
	m.Region = types.StringValue(region)
	m.StorageLimit = types.Int64Value(int64(objectStorageUser.QuotaGB))
	m.ZoneReplicationEnabled = types.BoolValue(objectStorageUser.ZoneReplicationEnabled)

	if objectStorageUser.S3Endpoints == nil {
		m.S3Endpoints = types.SetNull(types.StringType)
	} else {
		s3Endpoints, d := types.SetValueFrom(ctx, types.StringType, objectStorageUser.S3Endpoints)
		diags.Append(d...)
		m.S3Endpoints = s3Endpoints
	}

	if objectStorageUser.Tenant != nil {
		m.TenantID = types.StringValue(objectStorageUser.Tenant.ID)
	}

	return diags
}
