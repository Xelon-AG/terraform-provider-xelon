package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*isoResource)(nil)
	_ resource.ResourceWithConfigure   = (*isoResource)(nil)
	_ resource.ResourceWithImportState = (*isoResource)(nil)
)

// isoResource is the ISO resource implementation.
type isoResource struct {
	client *xelon.Client
}

// isoResourceModel maps the ISO resource schema data.
type isoResourceModel struct {
	Active      types.Bool   `tfsdk:"active"`
	CategoryID  types.Int64  `tfsdk:"category_id"`
	CloudID     types.String `tfsdk:"cloud_id"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	TenantID    types.String `tfsdk:"tenant_id"`
	URL         types.String `tfsdk:"url"`
}

func NewISOResource() resource.Resource {
	return &isoResource{}
}

func (r *isoResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_iso"
}

func (r *isoResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The ISO resource allows you to manage Xelon custom ISOs.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether ISO is active and can be used.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"category_id": schema.Int64Attribute{
				MarkdownDescription: "The category ID of the ISO.",
				Required:            true,
			},
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The ISO description.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the ISO.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the ISO.",
				Required:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the ISO belongs.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "The URL from which the ISO may be retrieved.",
				Required:            true,
			},
		},
	}
}

func (r *isoResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *isoResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data isoResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &xelon.ISOCreateRequest{
		CategoryID: int(data.CategoryID.ValueInt64()),
		CloudID:    data.CloudID.ValueString(),
		Name:       data.Name.ValueString(),
		URL:        data.URL.ValueString(),
	}
	if data.Description.ValueString() != "" {
		createRequest.Description = data.Description.ValueString()
	}
	if data.TenantID.ValueString() != "" {
		createRequest.TenantID = data.TenantID.ValueString()
	}
	tflog.Debug(ctx, "Creating ISO", map[string]any{"payload": createRequest})
	iso, _, err := r.client.ISOs.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create ISO", err.Error())
		return
	}
	tflog.Debug(ctx, "Created ISO", map[string]any{"data": iso})

	isoID := iso.ID
	tflog.Info(ctx, "Waiting for ISO to be active", map[string]any{"iso_id": isoID})
	err = helper.WaitISOActive(ctx, r.client, isoID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for ISO to be active", err.Error())
		return
	}
	tflog.Info(ctx, "ISO is active", map[string]any{"iso_id": isoID})

	tflog.Debug(ctx, "Getting ISO with enriched properties", map[string]any{"iso_id": isoID})
	iso, _, err = r.client.ISOs.Get(ctx, isoID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get ISO", err.Error())
		return
	}
	tflog.Debug(ctx, "Got ISO with enriched properties", map[string]any{"data": iso})

	// map response body to attributes
	data.Active = types.BoolValue(iso.Active)
	data.CloudID = types.StringValue(iso.Cloud.ID)
	data.Description = types.StringValue(iso.Description)
	data.ID = types.StringValue(iso.ID)
	data.Name = types.StringValue(iso.Name)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *isoResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data isoResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	isoID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting ISO", map[string]any{"iso_id": isoID})
	iso, resp, err := r.client.ISOs.Get(ctx, isoID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the tag is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get ISO", err.Error())
		return
	}
	tflog.Debug(ctx, "Got ISO", map[string]any{"data": iso})

	// map response body to attributes
	data.Active = types.BoolValue(iso.Active)
	data.CloudID = types.StringValue(iso.Cloud.ID)
	data.Description = types.StringValue(iso.Description)
	data.ID = types.StringValue(iso.ID)
	data.Name = types.StringValue(iso.Name)
}

func (r *isoResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data isoResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	isoID := data.ID.ValueString()
	updateRequest := &xelon.ISOUpdateRequest{
		CategoryID:  int(data.CategoryID.ValueInt64()),
		Description: data.Description.ValueString(),
		Name:        data.Name.ValueString(),
	}

	tflog.Debug(ctx, "Updating ISO", map[string]any{"iso_id": isoID, "payload": updateRequest})
	iso, _, err := r.client.ISOs.Update(ctx, isoID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update ISO", err.Error())
		return
	}
	tflog.Debug(ctx, "Updated ISO", map[string]any{"iso_id": isoID, "data": iso})

	// map response body to attributes
	data.Description = types.StringValue(iso.Description)
	data.Name = types.StringValue(iso.Name)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *isoResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data isoResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	isoID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting ISO", map[string]any{"iso_id": isoID})
	resp, err := r.client.ISOs.Delete(ctx, isoID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the ISO is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to delete ISO", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted ISO", map[string]any{"iso_id": isoID})
}

func (r *isoResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
