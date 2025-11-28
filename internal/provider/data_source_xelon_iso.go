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
	_ datasource.DataSource              = (*isoDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*isoDataSource)(nil)
)

// isoDataSource is the ISO data source implementation.
type isoDataSource struct {
	client *xelon.Client
}

// isoDataSourceModel maps the ISO datasource schema data.
type isoDataSourceModel struct {
	Active      types.Bool   `tfsdk:"active"`
	CloudID     types.String `tfsdk:"cloud_id"`
	Description types.String `tfsdk:"description"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
}

func NewISODataSource() datasource.DataSource {
	return &isoDataSource{}
}

func (d *isoDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_iso"
}

func (d *isoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The ISO data source provides information about an existing ISO.
`,
		Attributes: map[string]schema.Attribute{
			"active": schema.BoolAttribute{
				MarkdownDescription: "Whether ISO is active and can be used.",
				Computed:            true,
			},
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The ISO description.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the ISO.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the ISO.",
				Computed:            true,
				Optional:            true,
			},
		},
	}
}

func (d *isoDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *isoDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data isoDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	isoID := data.ID.ValueString()
	isoName := data.Name.ValueString()
	if isoID == "" && isoName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	if isoID != "" {
		tflog.Info(ctx, "Searching for ISO by ID", map[string]any{"iso_id": isoID})

		tflog.Debug(ctx, "Getting ISO", map[string]any{"iso_id": isoID})
		iso, resp, err := d.client.ISOs.Get(ctx, isoID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
			response.Diagnostics.AddError("Unable to get ISO", err.Error())
			return
		}
		tflog.Debug(ctx, "Got ISO", map[string]any{"data": iso, "iso_id": isoID})

		// if name is defined check that it's equal
		if isoName != "" && isoName != iso.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual ISO name are different: expected '%s', got '%s'.", isoName, iso.Name),
			)
			return
		}

		// map response body to attributes
		data.Active = types.BoolValue(iso.Active)
		data.CloudID = types.StringValue(iso.Cloud.ID)
		data.Description = types.StringValue(iso.Description)
		data.ID = types.StringValue(iso.ID)
		data.Name = types.StringValue(iso.Name)
	} else {
		tflog.Info(ctx, "Searching for ISO by name", map[string]any{"iso_name": isoName})

		tflog.Debug(ctx, "Getting ISOs", map[string]any{"iso_name": isoName})
		isos, _, err := d.client.ISOs.List(ctx, &xelon.ISOListOptions{Search: isoName})
		if err != nil {
			response.Diagnostics.AddError("Unable to search ISO by name", err.Error())
			return
		}
		tflog.Debug(ctx, "Got ISOs", map[string]any{"data": isos})

		if len(isos) == 0 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
		if len(isos) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v custom ISOs.", len(isos)),
			)
			return
		}

		iso := isos[0]
		// map response body to attributes
		data.Active = types.BoolValue(iso.Active)
		data.CloudID = types.StringValue(iso.Cloud.ID)
		data.Description = types.StringValue(iso.Description)
		data.ID = types.StringValue(iso.ID)
		data.Name = types.StringValue(iso.Name)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
