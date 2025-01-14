package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ datasource.DataSource              = (*networkDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*networkDataSource)(nil)
)

// networkDataSource is the network data source implementation.
type networkDataSource struct {
	client *xelon.Client
}

// networkDataSourceModel maps the network data source schema data.
type networkDataSourceModel struct {
	Clouds       []cloudDataSourceModel `tfsdk:"clouds"`
	DNSPrimary   types.String           `tfsdk:"dns_primary"`
	DNSSecondary types.String           `tfsdk:"dns_secondary"`
	ID           types.String           `tfsdk:"id"`
	Name         types.String           `tfsdk:"name"`
	Network      types.String           `tfsdk:"network"`
	SubnetSize   types.Int64            `tfsdk:"subnet_size"`
	Type         types.String           `tfsdk:"type"`
}

func NewNetworkDataSource() datasource.DataSource {
	return &networkDataSource{}
}

func (d *networkDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_network"
}

func (d *networkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "The network data source provides information about an existing network.",
		Attributes: map[string]schema.Attribute{
			"clouds": schema.SetNestedAttribute{
				MarkdownDescription: "The clouds of the network.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The ID of the cloud.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the network.",
							Computed:            true,
						},
					},
				},
			},
			"dns_primary": schema.StringAttribute{
				MarkdownDescription: "The primary DNS server address.",
				Computed:            true,
			},
			"dns_secondary": schema.StringAttribute{
				MarkdownDescription: "The secondary DNS server address.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the network.",
				Computed:            true,
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "The network definition.",
				Computed:            true,
			},
			"subnet_size": schema.Int64Attribute{
				MarkdownDescription: "The subnet size of the network.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the network (LAN or WAN).",
				Computed:            true,
			},
		},
	}
}

func (d *networkDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *networkDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data networkDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	networkID := data.ID.ValueString()
	if networkID == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" must be defined.`,
		)
		return
	}

	tflog.Debug(ctx, "Getting network by ID", map[string]any{"network_id": networkID})
	network, _, err := d.client.Networks.Get(ctx, networkID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get network", err.Error())
		return
	}
	tflog.Debug(ctx, "Got network by ID", map[string]any{"data": network, "network_id": networkID})

	var clouds []cloudDataSourceModel
	for _, cloud := range network.Clouds {
		clouds = append(clouds, cloudDataSourceModel{
			ID:   types.StringValue(cloud.ID),
			Name: types.StringValue(cloud.Name),
		})
	}

	// map response body to attributes
	data.Clouds = clouds
	data.DNSPrimary = types.StringValue(network.DNSPrimary)
	data.DNSSecondary = types.StringValue(network.DNSSecondary)
	data.ID = types.StringValue(network.ID)
	data.Name = types.StringValue(network.Name)
	data.Network = types.StringValue(network.Network)
	data.SubnetSize = types.Int64Value(int64(network.SubnetSize))
	data.Type = types.StringValue(network.Type)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
