package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*templateResource)(nil)
	_ resource.ResourceWithConfigure   = (*templateResource)(nil)
	_ resource.ResourceWithImportState = (*templateResource)(nil)
)

// templateResource is the template resource implementation.
type templateResource struct {
	client *xelon.Client
}

// templateResourceModel maps the template resource schema data.
type templateResourceModel struct {
	Category    types.String `tfsdk:"category"`
	CloudID     types.String `tfsdk:"cloud_id"`
	Description types.String `tfsdk:"description"`
	DeviceID    types.String `tfsdk:"device_id"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	TenantID    types.String `tfsdk:"tenant_id"`
	Type        types.String `tfsdk:"type"`
}

func NewTemplateResource() resource.Resource {
	return &templateResource{}
}

func (r *templateResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_template"
}

func (r *templateResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The template resource allows you to manage Xelon custom templates.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"category": schema.StringAttribute{
				MarkdownDescription: "The category of the template.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The template description.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the device from which the template will be created.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the template.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the template.",
				Required:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID of the source device.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the template.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *templateResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *templateResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data templateResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &xelon.TemplateCreateRequest{
		DeviceID:  data.DeviceID.ValueString(),
		Name:      data.Name.ValueString(),
		SendEmail: false,
		TenantID:  data.TenantID.ValueString(),
	}
	if data.Description.ValueString() != "" {
		createRequest.Description = data.Description.ValueString()
	}
	tflog.Debug(ctx, "Creating template", map[string]any{"payload": createRequest})
	template, _, err := r.client.Templates.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create template", err.Error())
		return
	}
	tflog.Debug(ctx, "Created template", map[string]any{"data": template})

	templateID := template.ID
	tflog.Info(ctx, "Waiting for template to be ready", map[string]any{"template_id": templateID})
	err = helper.WaitTemplateStateReady(ctx, r.client, templateID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for template to be ready", err.Error())
		return
	}
	tflog.Info(ctx, "Template is ready", map[string]any{"template_id": templateID})

	tflog.Debug(ctx, "Getting template with enriched properties", map[string]any{"template_id": templateID})
	template, _, err = r.client.Templates.Get(ctx, templateID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get template", err.Error())
		return
	}
	tflog.Debug(ctx, "Got template with enriched properties", map[string]any{"data": template})

	// map response body to attributes
	data.Category = types.StringValue(template.Category)
	data.CloudID = types.StringValue(template.CloudID)
	data.Description = types.StringValue(template.Description)
	data.ID = types.StringValue(template.ID)
	data.Name = types.StringValue(template.Name)
	data.Type = types.StringValue(template.Type)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *templateResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data templateResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	templateID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting template", map[string]any{"template_id": templateID})
	template, resp, err := r.client.Templates.Get(ctx, templateID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the template is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get template", err.Error())
		return
	}
	tflog.Debug(ctx, "Got template", map[string]any{"data": template})

	// map response body to attributes
	data.Category = types.StringValue(template.Category)
	data.CloudID = types.StringValue(template.CloudID)
	data.Description = types.StringValue(template.Description)
	data.ID = types.StringValue(template.ID)
	data.Name = types.StringValue(template.Name)
	data.Type = types.StringValue(template.Type)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *templateResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data templateResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	templateID := data.ID.ValueString()
	updateRequest := &xelon.TemplateUpdateRequest{
		Description: data.Description.ValueString(),
		Name:        data.Name.ValueString(),
	}
	tflog.Debug(ctx, "Updating template", map[string]any{"payload": updateRequest, "template_id": templateID})
	template, _, err := r.client.Templates.Update(ctx, templateID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update template", err.Error())
		return
	}
	tflog.Debug(ctx, "Updated template", map[string]any{"data": template})

	// map response body to attributes
	data.Category = types.StringValue(template.Category)
	data.CloudID = types.StringValue(template.CloudID)
	data.Description = types.StringValue(template.Description)
	data.ID = types.StringValue(template.ID)
	data.Name = types.StringValue(template.Name)
	data.Type = types.StringValue(template.Type)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *templateResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data templateResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	templateID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting template", map[string]any{"template_id": templateID})
	_, err := r.client.Templates.Delete(ctx, templateID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete template", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted template", map[string]any{"template_id": templateID})
}

func (r *templateResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
