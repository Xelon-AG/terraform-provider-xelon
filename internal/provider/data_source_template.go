package provider

import (
	"context"
	"fmt"
	"strconv"

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

// templateDataSource is the template datasource implementation.
type templateDataSource struct {
	client *xelon.Client
}

// templateDataSourceModel maps the template datasource schema data.
type templateDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	CloudID     types.Int64  `tfsdk:"cloud_id"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
	Category    types.String `tfsdk:"category"`
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
The template data source provides information about an existing Xelon template.

Templates are OS images that can be used to create devices (virtual machines). This data source allows you to look up templates by ID or name, and filter by operating system type and cloud.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the template.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The template name (exact match).",
				Computed:            true,
				Optional:            true,
			},
			"cloud_id": schema.Int64Attribute{
				MarkdownDescription: "Filter templates by cloud ID. If not specified, searches across all clouds.",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Filter by operating system type. Valid values: `Linux`, `Windows`. This is a client-side filter.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The template description.",
				Computed:            true,
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "The template category.",
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

	// Build API filter options for efficient server-side filtering
	opts := &xelon.TemplateListOptions{
		PerPage: 100, // Max per page for efficiency
		Sort:    "name",
	}

	// Server-side cloud ID filter
	if !data.CloudID.IsNull() {
		cloudIDFilter := fmt.Sprintf("%d", data.CloudID.ValueInt64())
		opts.CloudIdentifier = cloudIDFilter
		tflog.Debug(ctx, "Filtering by cloud ID on server", map[string]any{"cloud_id": cloudIDFilter})
	}

	// Server-side type filter
	if data.Type.ValueString() != "" {
		opts.Type = data.Type.ValueString()
		tflog.Debug(ctx, "Filtering by type on server", map[string]any{"type": opts.Type})
	}

	// Server-side search filter (if searching by name only)
	if templateName != "" && templateID == "" {
		opts.Search = templateName
		tflog.Debug(ctx, "Using server-side search", map[string]any{"search": templateName})
	}

	// Fetch templates with server-side filtering
	tflog.Debug(ctx, "Getting templates from v2 API with filters", map[string]any{
		"cloud_id": opts.CloudIdentifier,
		"type":     opts.Type,
		"search":   opts.Search,
	})

	allTemplates, _, err := d.client.Templates.List(ctx, opts)
	if err != nil {
		response.Diagnostics.AddError("Unable to list templates", err.Error())
		return
	}

	tflog.Info(ctx, "Got filtered templates from API", map[string]any{"count": len(allTemplates)})

	if len(allTemplates) == 0 {
		response.Diagnostics.AddError(
			"No templates found",
			"The API returned no templates matching your filters. Please check your search criteria.",
		)
		return
	}

	// Search for template by ID or name
	var template *xelon.TemplateV2
	for _, t := range allTemplates {
		if templateID == t.Identifier {
			// if name is defined check that it's equal
			if templateName != "" && templateName != t.Name {
				response.Diagnostics.AddError(
					"Ambiguous search result",
					fmt.Sprintf("Specified and actual template name are different: expected '%s', got '%s'.", templateName, t.Name),
				)
				return
			}
			template = &t
			break
		}
		if templateName == t.Name {
			// if id is defined check that it's equal
			if templateID != "" && templateID != t.Identifier {
				response.Diagnostics.AddError(
					"Ambiguous search result",
					fmt.Sprintf("Specified and actual template identifier are different: expected '%s', got '%s'.", templateID, t.Identifier),
				)
				return
			}
			template = &t
			break
		}
	}

	if template == nil {
		response.Diagnostics.AddError(
			"No search results",
			"No template found matching the specified criteria. Please refine your search.",
		)
		return
	}

	// map response body to attributes
	data.ID = types.StringValue(template.Identifier)
	data.Name = types.StringValue(template.Name)
	data.Type = types.StringValue(template.Type)
	data.Description = types.StringValue(template.Description)
	data.Category = types.StringValue(template.Category)
	if template.CloudIdentifier != "" {
		cloudIDInt, err := strconv.ParseInt(template.CloudIdentifier, 10, 64)
		if err == nil {
			data.CloudID = types.Int64Value(cloudIDInt)
		}
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
