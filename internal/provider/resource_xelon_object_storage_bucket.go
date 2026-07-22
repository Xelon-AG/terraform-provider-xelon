package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                   = (*objectStorageBucketResource)(nil)
	_ resource.ResourceWithConfigure      = (*objectStorageBucketResource)(nil)
	_ resource.ResourceWithImportState    = (*objectStorageBucketResource)(nil)
	_ resource.ResourceWithValidateConfig = (*objectStorageBucketResource)(nil)
)

const objectStorageBucketObjectLockRetentionDaysMaximum int64 = 36500

// objectStorageBucketResource is the object storage bucket resource implementation.
type objectStorageBucketResource struct {
	client *xelon.Client
}

// objectStorageBucketResourceModel maps the object storage bucket resource schema data.
type objectStorageBucketResourceModel struct {
	CreatedAt                types.String `tfsdk:"created_at"`
	ID                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	ObjectLockEnabled        types.Bool   `tfsdk:"object_lock_enabled"`
	ObjectLockRetentionDays  types.Int64  `tfsdk:"object_lock_retention_days"`
	ObjectStorageUserID      types.String `tfsdk:"user_id"`
	RegionReplicationEnabled types.Bool   `tfsdk:"region_replication_enabled"`
	S3Endpoints              types.Set    `tfsdk:"s3_endpoints"` // []types.String
	TenantID                 types.String `tfsdk:"tenant_id"`
	VersioningEnabled        types.Bool   `tfsdk:"versioning_enabled"`
}

func NewObjectStorageBucketResource() resource.Resource {
	return &objectStorageBucketResource{}
}

func (r *objectStorageBucketResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_object_storage_bucket"
}

func (r *objectStorageBucketResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The object storage bucket resource allows you to manage an S3-compatible bucket in Xelon Object Storage.

A bucket belongs to an object storage user and is created in that user's region. Versioning can be managed through this resource.
Object Lock is optional and defaults to disabled. Object Lock can only be enabled when a bucket is created. When Object Lock is enabled, versioning is required, and bucket deletion fails while retained objects remain.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the bucket was created in RFC3339 format.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The stable ID of the bucket.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the bucket. Changing this value requires replacing the bucket.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"object_lock_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether Object Lock is enabled for the bucket. Object Lock requires versioning, " +
					"can only be enabled when the bucket is created, and cannot be disabled. Changing this value requires replacing the bucket.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"object_lock_retention_days": schema.Int64Attribute{
				MarkdownDescription: "The default Object Lock retention period, in days, applied to new object versions. " +
					"This value can only be configured when Object Lock is enabled. When omitted, the bucket has no default retention period. " +
					"Changing this value requires replacing the bucket.",
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.Between(1, objectStorageBucketObjectLockRetentionDaysMaximum),
				},
			},
			"region_replication_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether replication is enabled for the bucket's Object Storage region.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"s3_endpoints": schema.SetAttribute{
				MarkdownDescription: "S3-compatible endpoint URLs for the bucket's region.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the tenant that owns the bucket.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the object storage user that owns the bucket.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"versioning_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether bucket versioning is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
	}
}

func (r *objectStorageBucketResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *objectStorageBucketResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data objectStorageBucketResourceModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	userID := data.ObjectStorageUserID.ValueString()
	bucketName := data.Name.ValueString()

	objectLockRetentionDays := 0
	if !data.ObjectLockRetentionDays.IsNull() &&
		!data.ObjectLockRetentionDays.IsUnknown() {
		objectLockRetentionDays = int(data.ObjectLockRetentionDays.ValueInt64())
	}

	createRequest := &xelon.ObjectStorageBucketCreateRequest{
		Name:                    bucketName,
		ObjectLockEnabled:       data.ObjectLockEnabled.ValueBool(),
		ObjectLockRetentionDays: objectLockRetentionDays,
		ObjectStorageUserID:     userID,
		VersioningEnabled:       data.VersioningEnabled.ValueBool(),
	}

	tflog.Debug(ctx, "creating object storage bucket", map[string]any{
		"name":    bucketName,
		"user_id": userID,
	})

	tflog.Trace(ctx, "creating object storage bucket via API", map[string]any{
		"name":    bucketName,
		"user_id": userID,
		"payload": createRequest,
	})
	createdBucket, _, err := r.client.ObjectStorages.CreateBucket(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create object storage bucket", err.Error())
		return
	}
	tflog.Trace(ctx, "received object storage bucket from API", map[string]any{"data": createdBucket})

	tflog.Debug(ctx, "created object storage bucket", map[string]any{
		"id":      createdBucket.ID,
		"name":    createdBucket.Name,
		"user_id": userID,
	})

	response.Diagnostics.Append(data.fromAPI(ctx, createdBucket, userID)...)
	if response.Diagnostics.HasError() {
		return
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *objectStorageBucketResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data objectStorageBucketResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	bucketName := data.Name.ValueString()
	userID := data.ObjectStorageUserID.ValueString()

	tflog.Debug(ctx, "reading object storage bucket", map[string]any{
		"name":    bucketName,
		"user_id": userID,
	})

	tflog.Trace(ctx, "reading object storage bucket via API", map[string]any{
		"name":    bucketName,
		"user_id": userID,
	})
	bucket, resp, err := r.client.ObjectStorages.GetBucket(ctx, bucketName, userID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "object storage bucket not found; removing from state", map[string]any{
				"name":    bucketName,
				"user_id": userID,
			})
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read object storage bucket", err.Error())
		return
	}
	tflog.Trace(ctx, "received object storage bucket from API", map[string]any{"data": bucket})

	response.Diagnostics.Append(data.fromAPI(ctx, bucket, userID)...)
	if response.Diagnostics.HasError() {
		return
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *objectStorageBucketResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan objectStorageBucketResourceModel
	var state objectStorageBucketResourceModel

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

	userID := plan.ObjectStorageUserID.ValueString()
	bucketID := state.ID.ValueString()
	bucketName := state.Name.ValueString()
	versioningChanged := state.VersioningEnabled.ValueBool() != plan.VersioningEnabled.ValueBool()

	tflog.Debug(ctx, "updating object storage bucket", map[string]any{
		"id":      bucketID,
		"name":    bucketName,
		"user_id": userID,
	})

	if versioningChanged {
		tflog.Trace(ctx, "updating object storage bucket versioning via API", map[string]any{
			"name":               bucketName,
			"user_id":            userID,
			"versioning_enabled": plan.VersioningEnabled.ValueBool(),
		})
		_, err := r.client.ObjectStorages.UpdateBucketVersioning(ctx, bucketName, userID, &xelon.ObjectStorageBucketVersioningUpdateRequest{
			VersioningEnabled: plan.VersioningEnabled.ValueBool(),
		})
		if err != nil {
			response.Diagnostics.AddError("Unable to update object storage bucket versioning", err.Error())
			return
		}

		tflog.Debug(ctx, "updated object storage bucket versioning", map[string]any{
			"name":               bucketName,
			"user_id":            userID,
			"versioning_enabled": plan.VersioningEnabled.ValueBool(),
		})
	}

	tflog.Debug(ctx, "refreshing object storage bucket state after update", map[string]any{
		"name":    bucketName,
		"user_id": userID,
	})

	tflog.Trace(ctx, "reading object storage bucket via API (updated bucket refresh)", map[string]any{
		"name":    bucketName,
		"user_id": userID,
	})
	bucket, _, err := r.client.ObjectStorages.GetBucket(ctx, bucketName, userID)
	if err != nil {
		response.Diagnostics.AddError("Unable to read object storage bucket", err.Error())
		return
	}
	tflog.Trace(ctx, "received object storage bucket from API", map[string]any{"data": bucket})

	response.Diagnostics.Append(plan.fromAPI(ctx, bucket, userID)...)
	if response.Diagnostics.HasError() {
		return
	}

	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}

func (r *objectStorageBucketResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data objectStorageBucketResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	userID := data.ObjectStorageUserID.ValueString()
	bucketName := data.Name.ValueString()

	tflog.Debug(ctx, "deleting object storage bucket", map[string]any{
		"name":    bucketName,
		"user_id": userID,
	})

	tflog.Trace(ctx, "deleting object storage bucket via API", map[string]any{
		"name":    bucketName,
		"user_id": userID,
	})
	resp, err := r.client.ObjectStorages.DeleteBucket(ctx, bucketName, userID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "object storage bucket already absent", map[string]any{
				"name":    bucketName,
				"user_id": userID,
			})
			return
		}
		response.Diagnostics.AddError("Unable to delete object storage bucket", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted object storage bucket", map[string]any{
		"name":    bucketName,
		"user_id": userID,
	})
}

func (r *objectStorageBucketResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	userID, bucketName, err := parseObjectStorageBucketImportID(request.ID)
	if err != nil {
		response.Diagnostics.AddError("Invalid import identifier", "Expected format: <user-id>/<bucket-name>")
		return
	}

	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("name"), bucketName)...)
}

func (r *objectStorageBucketResource) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var data objectStorageBucketResourceModel

	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	objectLockEnabled, objectLockKnown := resolveBoolWithDefaultFalse(data.ObjectLockEnabled)
	versioningEnabled, versioningKnown := resolveBoolWithDefaultFalse(data.VersioningEnabled)
	retentionConfigured := !data.ObjectLockRetentionDays.IsNull() && !data.ObjectLockRetentionDays.IsUnknown()

	if objectLockKnown && versioningKnown && objectLockEnabled && !versioningEnabled {
		response.Diagnostics.AddAttributeError(
			path.Root("versioning_enabled"),
			"Object Lock requires versioning",
			`Attribute "versioning_enabled" must be true when "object_lock_enabled" is true.`,
		)
	}

	if retentionConfigured && objectLockKnown && !objectLockEnabled {
		response.Diagnostics.AddAttributeError(
			path.Root("object_lock_retention_days"),
			"Object Lock retention requires Object Lock",
			`Attribute "object_lock_retention_days" can only be configured when "object_lock_enabled" is true.`,
		)
	}
}

func (m *objectStorageBucketResourceModel) fromAPI(ctx context.Context, bucket *xelon.ObjectStorageBucket, objectStorageUserID string) diag.Diagnostics {
	var diags diag.Diagnostics

	m.CreatedAt = helper.FormatTimeRFC3339(bucket.CreatedAt)
	m.ID = types.StringValue(bucket.ID)
	m.Name = types.StringValue(bucket.Name)
	m.ObjectLockEnabled = types.BoolValue(bucket.ObjectLockEnabled)
	if bucket.ObjectLockRetentionDays > 0 {
		m.ObjectLockRetentionDays = types.Int64Value(int64(bucket.ObjectLockRetentionDays))
	} else {
		m.ObjectLockRetentionDays = types.Int64Null()
	}
	m.ObjectStorageUserID = types.StringValue(objectStorageUserID)
	m.RegionReplicationEnabled = types.BoolValue(bucket.RegionReplicationEnabled)
	m.VersioningEnabled = types.BoolValue(bucket.VersioningEnabled)

	if bucket.S3Endpoints == nil {
		m.S3Endpoints = types.SetNull(types.StringType)
	} else {
		s3Endpoints, d := types.SetValueFrom(ctx, types.StringType, bucket.S3Endpoints)
		diags.Append(d...)
		m.S3Endpoints = s3Endpoints
	}

	if bucket.Tenant != nil {
		m.TenantID = types.StringValue(bucket.Tenant.ID)
	} else {
		m.TenantID = types.StringNull()
	}

	return diags
}

func parseObjectStorageBucketImportID(importID string) (string, string, error) {
	userID, bucketName, ok := strings.Cut(importID, "/")
	if !ok || userID == "" || bucketName == "" {
		return "", "", errors.New("invalid import identifier")
	}

	return userID, bucketName, nil
}

func resolveBoolWithDefaultFalse(value types.Bool) (resolved bool, known bool) {
	if value.IsUnknown() {
		return false, false
	}

	if value.IsNull() {
		return false, true
	}

	return value.ValueBool(), true
}
