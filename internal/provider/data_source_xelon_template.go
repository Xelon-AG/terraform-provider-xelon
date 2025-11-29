package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ datasource.DataSource              = (*templateDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*templateDataSource)(nil)
)

// templateDataSource is the template data source implementation.
type templateDataSource struct {
	client *xelon.Client
}

// templateDataSourceModel maps the template datasource schema data.
type templateDataSourceModel struct {
	Category    types.String `tfsdk:"category"`
	CloudID     types.String `tfsdk:"cloud_id"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
}

func NewTemplateDataSource() datasource.DataSource {
	return &templateDataSource{}
}

func (d *templateDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_template"
}

func (d *templateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The template data source provides information about an existing template.
`,
		Attributes: map[string]schema.Attribute{
			"category": schema.StringAttribute{
				MarkdownDescription: "The category of the template.",
				Computed:            true,
			},
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The template description.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the template.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the template.",
				Computed:            true,
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the template.",
				Computed:            true,
			},
		},
	}
}

func (d *templateDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *templateDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data templateDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	templateID := data.ID.ValueString()
	templateName := data.Name.ValueString()
	if templateID == "" && templateName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	if templateID != "" {
		tflog.Info(ctx, "Searching for template by ID", map[string]any{"template_id": templateID})

		tflog.Debug(ctx, "Getting template", map[string]any{"template_id": templateID})
		template, resp, err := d.client.Templates.Get(ctx, templateID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
			response.Diagnostics.AddError("Unable to get template", err.Error())
			return
		}
		tflog.Debug(ctx, "Got template", map[string]any{"data": template, "template_id": templateID})

		// if name is defined check that it's equal
		if templateName != "" && templateName != template.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual template name are different: expected '%s', got '%s'.", templateName, template.Name),
			)
			return
		}

		// map response body to attributes
		data.Category = types.StringValue(template.Category)
		data.CloudID = types.StringValue(template.CloudID)
		data.Description = types.StringValue(template.Description)
		data.ID = types.StringValue(template.ID)
		data.Name = types.StringValue(template.Name)
		data.Type = types.StringValue(template.Type)
	} else {
		tflog.Info(ctx, "Searching for template by name", map[string]any{"template_name": templateName})

		tflog.Debug(ctx, "Getting templates", map[string]any{"template_name": templateName})
		templates, _, err := d.client.Templates.List(ctx, &xelon.TenantListOptions{Search: templateName})
		if err != nil {
			response.Diagnostics.AddError("Unable to search template by name", err.Error())
			return
		}
		tflog.Debug(ctx, "Got templates", map[string]any{"data": templates})

		if len(templates) == 0 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
		if len(templates) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v templates.", len(templates)),
			)
			return
		}

		template := templates[0]
		// map response body to attributes
		data.Category = types.StringValue(template.Category)
		data.CloudID = types.StringValue(template.CloudID)
		data.Description = types.StringValue(template.Description)
		data.ID = types.StringValue(template.ID)
		data.Name = types.StringValue(template.Name)
		data.Type = types.StringValue(template.Type)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
