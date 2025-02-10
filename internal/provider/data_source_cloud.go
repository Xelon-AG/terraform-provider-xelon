package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ datasource.DataSource              = (*cloudDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*cloudDataSource)(nil)
)

// cloudDataSource is the cloud datasource implementation.
type cloudDataSource struct {
	client *xelon.Client
}

// cloudDataSourceModel maps the cloud datasource schema data.
type cloudDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewCloudDataSource() datasource.DataSource {
	return &cloudDataSource{}
}

func (d *cloudDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_cloud"
}

func (d *cloudDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The cloud data source provides information about an existing cloud.

Xelon cloud is data center where your resources are hosted.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The cloud name.",
				Computed:            true,
				Optional:            true,
			},
		},
	}
}
func (d *cloudDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *cloudDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data cloudDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	cloudID := data.ID.ValueString()
	cloudName := data.Name.ValueString()
	if cloudID == "" && cloudName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	tflog.Debug(ctx, "Getting clouds")
	clouds, _, err := d.client.Clouds.List(ctx, nil)
	if err != nil {
		response.Diagnostics.AddError("Unable to list clouds", err.Error())
		return
	}
	tflog.Debug(ctx, "Got clouds", map[string]any{"data": clouds})

	var cloud *xelon.Cloud
	for _, c := range clouds {
		if cloudID == c.ID {
			// if name is defined check that it's equal
			if cloudName != "" && cloudName != c.Name {
				response.Diagnostics.AddError(
					"Ambiguous search result",
					fmt.Sprintf("Specified and actual cloud name are different: expected '%s', got '%s'.", cloudName, c.Name),
				)
				return
			}
			cloud = &c
			break
		}
		if cloudName == c.Name {
			// if id is defined check that it's equal
			if cloudID != "" && cloudID != c.ID {
				response.Diagnostics.AddError(
					"Ambiguous search result",
					fmt.Sprintf("Specified and actual cloud identifier are different: expected '%s', got '%s'.", cloudID, c.ID),
				)
				return
			}
			cloud = &c
			break
		}
	}

	if cloud == nil {
		response.Diagnostics.AddError("No search results", "Please refine your search.")
		return
	}

	// map response body to attributes
	data.ID = types.StringValue(cloud.ID)
	data.Name = types.StringValue(cloud.Name)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
