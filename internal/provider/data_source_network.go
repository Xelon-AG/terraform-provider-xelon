package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

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
	CloudID      types.Int64                  `tfsdk:"cloud_id"`
	DNSPrimary   types.String                 `tfsdk:"dns_primary"`
	DNSSecondary types.String                 `tfsdk:"dns_secondary"`
	Filter       networkDataSourceFilterModel `tfsdk:"filter"`
	ID           types.String                 `tfsdk:"id"`
	Name         types.String                 `tfsdk:"name"`
	Netmask      types.String                 `tfsdk:"netmask"`
	Network      types.String                 `tfsdk:"network"`
	NetworkID    types.Int64                  `tfsdk:"network_id"`
}

type networkDataSourceFilterModel struct {
	NetworkID types.Int64 `tfsdk:"network_id"`
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
			"cloud_id": schema.Int64Attribute{
				MarkdownDescription: "The cloud ID of the organization (tenant).",
				Computed:            true,
			},

			"dns_primary": schema.StringAttribute{
				MarkdownDescription: "The primary DNS server address.",
				Computed:            true,
			},

			"dns_secondary": schema.StringAttribute{
				MarkdownDescription: "The secondary DNS server address.",
				Computed:            true,
			},

			"filter": schema.SingleNestedAttribute{
				MarkdownDescription: "The filter specifies the criteria to retrieve a single network. " +
					"The retrieval will fail if the criteria match more than one item.",
				Required: true,
				Attributes: map[string]schema.Attribute{
					"network_id": schema.Int64Attribute{
						MarkdownDescription: "The ID of the specific network, must be a positive number.",
						Optional:            true,
						Validators: []validator.Int64{
							// network id must be a positive number
							int64validator.AtLeast(0),
						},
					},
				},
			},

			// id attribute is required for acceptance testing
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network.",
				Computed:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the network.",
				Computed:            true,
			},

			"netmask": schema.StringAttribute{
				MarkdownDescription: "The netmask of the network.",
				Computed:            true,
			},

			"network": schema.StringAttribute{
				MarkdownDescription: "A /24 network.",
				Computed:            true,
			},

			"network_id": schema.Int64Attribute{
				MarkdownDescription: "The ID of the specific network",
				Computed:            true,
			},
		},
	}
}

func (d *networkDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	d.client = request.ProviderData.(*xelon.Client)
}

func (d *networkDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data networkDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	tenant, _, err := d.client.Tenants.GetCurrent(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to fetch current tenant", err.Error())
		return
	}

	// at the moment network_id is mandatory
	if data.Filter.NetworkID.IsNull() {
		response.Diagnostics.AddAttributeError(
			path.Root("filter"),
			"Invalid Attribute Value",
			"Attribute filter.network_id must be set.",
		)
		return
	}

	n, _, err := d.client.Networks.Get(ctx, tenant.TenantID, int(data.Filter.NetworkID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Unable to get network info", err.Error())
		return
	}

	network := n.Details
	data.CloudID = types.Int64Value(int64(n.CloudID))
	data.DNSPrimary = types.StringValue(network.DNSPrimary)
	data.DNSSecondary = types.StringValue(network.DNSSecondary)
	data.ID = types.StringValue(strconv.Itoa(network.ID))
	data.Name = types.StringValue(network.Name)
	data.Netmask = types.StringValue(network.Netmask)
	data.Network = types.StringValue(network.Network)
	data.NetworkID = types.Int64Value(int64(network.ID))

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
