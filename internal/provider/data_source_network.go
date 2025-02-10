package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	Gateway      types.String           `tfsdk:"gateway"`
	ID           types.String           `tfsdk:"id"`
	Name         types.String           `tfsdk:"name"`
	Network      types.String           `tfsdk:"network"`
	SubnetSize   types.Int64            `tfsdk:"subnet_size"`
	TenantID     types.String           `tfsdk:"tenant_id"`
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
							MarkdownDescription: "The name of the cloud.",
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
			"gateway": schema.StringAttribute{
				MarkdownDescription: "The default gateway address.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The network name.",
				Computed:            true,
				Optional:            true,
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "The network definition.",
				Computed:            true,
			},
			"subnet_size": schema.Int64Attribute{
				MarkdownDescription: "The subnet size of the network.",
				Computed:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the network belongs.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the network (`LAN` or `WAN`).",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"LAN", "WAN"}...),
				},
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
	networkName := data.Name.ValueString()
	if networkID == "" && networkName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	if networkID != "" {
		tflog.Info(ctx, "Searching for network by ID", map[string]any{"network_id": networkID})

		tflog.Debug(ctx, "Getting network", map[string]any{"network_id": networkID})
		network, resp, err := d.client.Networks.Get(ctx, networkID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
			response.Diagnostics.AddError("Unable to get network", err.Error())
			return
		}
		tflog.Debug(ctx, "Got network", map[string]any{"data": network, "network_id": networkID})

		// if name is defined check that it's equal
		if networkName != "" && networkName != network.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual network name are different: expected '%s', got '%s'.", networkName, network.Name),
			)
			return
		}

		// map response body to attributes
		var clouds []cloudDataSourceModel
		for _, cloud := range network.Clouds {
			clouds = append(clouds, cloudDataSourceModel{
				ID:   types.StringValue(cloud.ID),
				Name: types.StringValue(cloud.Name),
			})
		}
		data.Clouds = clouds
		data.DNSPrimary = types.StringValue(network.DNSPrimary)
		data.DNSSecondary = types.StringValue(network.DNSSecondary)
		data.Gateway = types.StringValue(network.Gateway)
		data.ID = types.StringValue(network.ID)
		data.Name = types.StringValue(network.Name)
		data.Network = types.StringValue(network.Network)
		data.SubnetSize = types.Int64Value(int64(network.SubnetSize))
		if network.Owner != nil {
			data.TenantID = types.StringValue(network.Owner.ID)
		}
		data.Type = types.StringValue(network.Type)
	} else {
		tflog.Info(ctx, "Searching for network by name", map[string]any{"network_name": networkName})

		tflog.Debug(ctx, "Getting networks", map[string]any{"network_id": networkID})
		networks, _, err := d.client.Networks.List(ctx, &xelon.NetworkListOptions{Search: networkName})
		if err != nil {
			response.Diagnostics.AddError("Unable to search networks by name", err.Error())
			return
		}
		tflog.Debug(ctx, "Got networks", map[string]any{"data": networks})

		if len(networks) == 0 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
		if len(networks) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v networks.", len(networks)),
			)
			return
		}

		network := &networks[0]
		// enrich data because not all fields are exposed via list API
		tflog.Debug(ctx, "Getting network", map[string]any{"network_id": network.ID})
		network, _, err = d.client.Networks.Get(ctx, network.ID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get network", err.Error())
			return
		}
		tflog.Debug(ctx, "Got network", map[string]any{"data": network})

		// map response body to attributes
		var clouds []cloudDataSourceModel
		for _, cloud := range network.Clouds {
			clouds = append(clouds, cloudDataSourceModel{
				ID:   types.StringValue(cloud.ID),
				Name: types.StringValue(cloud.Name),
			})
		}
		data.Clouds = clouds
		data.DNSPrimary = types.StringValue(network.DNSPrimary)
		data.DNSSecondary = types.StringValue(network.DNSSecondary)
		data.Gateway = types.StringValue(network.Gateway)
		data.ID = types.StringValue(network.ID)
		data.Name = types.StringValue(network.Name)
		data.Network = types.StringValue(network.Network)
		data.SubnetSize = types.Int64Value(int64(network.SubnetSize))
		if network.Owner != nil {
			data.TenantID = types.StringValue(network.Owner.ID)
		}
		data.Type = types.StringValue(network.Type)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
