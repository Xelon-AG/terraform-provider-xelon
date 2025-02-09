package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*networkResource)(nil)
	_ resource.ResourceWithConfigure   = (*networkResource)(nil)
	_ resource.ResourceWithImportState = (*networkResource)(nil)
)

// networkResource is the network resource implementation.
type networkResource struct {
	client *xelon.Client
}

// networkResourceModel maps the network resource schema data.
type networkResourceModel struct {
	CloudID      types.String `tfsdk:"cloud_id"`
	DNSPrimary   types.String `tfsdk:"dns_primary"`
	DNSSecondary types.String `tfsdk:"dns_secondary"`
	Gateway      types.String `tfsdk:"gateway"`
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Network      types.String `tfsdk:"network"`
	NetworkSpeed types.Int64  `tfsdk:"network_speed"`
	SubnetSize   types.Int64  `tfsdk:"subnet_size"`
	TenantID     types.String `tfsdk:"tenant_id"`
	Type         types.String `tfsdk:"type"`
}

func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

func (r *networkResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_network"
}

func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The network resource allows you to manage Xelon networks.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Required:            true,
			},
			"dns_primary": schema.StringAttribute{
				MarkdownDescription: "The primary DNS server address.",
				Required:            true,
			},
			"dns_secondary": schema.StringAttribute{
				MarkdownDescription: "The secondary DNS server address.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gateway": schema.StringAttribute{
				MarkdownDescription: "The default gateway address.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The network name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"network": schema.StringAttribute{
				MarkdownDescription: "The network definition.",
				Required:            true,
			},
			"network_speed": schema.Int64Attribute{
				MarkdownDescription: "The speed of the network in MBit. Must be one of `1000` or `10000`.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.OneOf([]int64{1_000, 10_000}...),
				},
			},
			"subnet_size": schema.Int64Attribute{
				MarkdownDescription: "The subnet size of the network.",
				Required:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the network belongs.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The network type. Must be one of `LAN` or `WAN`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"LAN", "WAN"}...),
				},
			},
		},
	}
}

func (r *networkResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *networkResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data networkResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Type.ValueString() == "LAN" {
		createRequest := &xelon.NetworkLANCreateRequest{
			CloudID:      data.CloudID.ValueString(),
			DNSPrimary:   data.DNSPrimary.ValueString(),
			Gateway:      data.Gateway.ValueString(),
			Name:         data.Name.ValueString(),
			Network:      data.Network.ValueString(),
			NetworkSpeed: int(data.NetworkSpeed.ValueInt64()),
			SubnetSize:   int(data.SubnetSize.ValueInt64()),
		}
		if data.DNSSecondary.ValueString() != "" {
			createRequest.DNSSecondary = data.DNSSecondary.ValueString()
		}
		if data.TenantID.ValueString() != "" {
			createRequest.TenantID = data.TenantID.ValueString()
		}
		tflog.Debug(ctx, "Creating LAN network", map[string]any{"payload": createRequest})
		network, _, err := r.client.Networks.CreateLAN(ctx, createRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to create LAN network", err.Error())
			return
		}
		tflog.Debug(ctx, "Created LAN network", map[string]any{"data": network})

		// enrich data because after create not all fields are exposed via API
		tflog.Debug(ctx, "Getting new created LAN network", map[string]any{"network_id": network.ID})
		network, _, err = r.client.Networks.Get(ctx, network.ID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get LAN network", err.Error())
			return
		}
		tflog.Debug(ctx, "Got new created LAN network", map[string]any{"data": network})

		// map response body to attributes
		data.CloudID = types.StringValue(network.Clouds[0].ID)
		data.DNSPrimary = types.StringValue(network.DNSPrimary)
		data.DNSSecondary = types.StringValue(network.DNSSecondary)
		data.Gateway = types.StringValue(network.Gateway)
		data.ID = types.StringValue(network.ID)
		data.Name = types.StringValue(network.Name)
		data.Network = types.StringValue(network.Network)
		data.NetworkSpeed = types.Int64Value(int64(network.NetworkSpeed))
		data.SubnetSize = types.Int64Value(int64(network.SubnetSize))
		if network.Owner != nil {
			data.TenantID = types.StringValue(network.Owner.ID)
		}
		data.Type = types.StringValue(network.Type)
	}
	if data.Type.ValueString() == "WAN" {
		tflog.Info(ctx, "Creating WAN network", map[string]any{"type": data.Type.ValueString()})
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *networkResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data networkResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	networkID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting network", map[string]any{"network_id": networkID})
	network, _, err := r.client.Networks.Get(ctx, networkID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get LAN network", err.Error())
		return
	}
	tflog.Debug(ctx, "Got network", map[string]any{"data": network})

	// map response body to attributes
	data.CloudID = types.StringValue(network.Clouds[0].ID)
	data.DNSPrimary = types.StringValue(network.DNSPrimary)
	data.DNSSecondary = types.StringValue(network.DNSSecondary)
	data.Gateway = types.StringValue(network.Gateway)
	data.ID = types.StringValue(network.ID)
	data.Name = types.StringValue(network.Name)
	data.Network = types.StringValue(network.Network)
	data.NetworkSpeed = types.Int64Value(int64(network.NetworkSpeed))
	data.SubnetSize = types.Int64Value(int64(network.SubnetSize))
	if network.Owner != nil {
		data.TenantID = types.StringValue(network.Owner.ID)
	}
	data.Type = types.StringValue(network.Type)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *networkResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data networkResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Type.ValueString() == "LAN" {
		networkID := data.ID.ValueString()
		updateRequest := &xelon.NetworkLANUpdateRequest{
			DNSPrimary:   data.DNSPrimary.ValueString(),
			DNSSecondary: data.DNSSecondary.ValueString(),
			Gateway:      data.Gateway.ValueString(),
			Name:         data.Name.ValueString(),
			Network:      data.Network.ValueString(),
			NetworkSpeed: int(data.NetworkSpeed.ValueInt64()),
		}
		tflog.Debug(ctx, "Updating LAN network", map[string]any{"network_id": networkID, "payload": updateRequest})
		network, _, err := r.client.Networks.UpdateLAN(ctx, networkID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update LAN network", err.Error())
			return
		}
		tflog.Debug(ctx, "Updated LAN network", map[string]any{"data": network})

		// map response body to attributes
		data.CloudID = types.StringValue(network.Clouds[0].ID)
		data.DNSPrimary = types.StringValue(network.DNSPrimary)
		data.DNSSecondary = types.StringValue(network.DNSSecondary)
		data.Gateway = types.StringValue(network.Gateway)
		data.ID = types.StringValue(network.ID)
		data.Name = types.StringValue(network.Name)
		data.Network = types.StringValue(network.Network)
		data.NetworkSpeed = types.Int64Value(int64(network.NetworkSpeed))
		data.SubnetSize = types.Int64Value(int64(network.SubnetSize))
		if network.Owner != nil {
			data.TenantID = types.StringValue(network.Owner.ID)
		}
		data.Type = types.StringValue(network.Type)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *networkResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data networkResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	networkID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting network", map[string]any{"network_id": networkID})
	_, err := r.client.Networks.Delete(ctx, networkID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete network", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted network", map[string]any{"network_id": networkID})
}

func (r *networkResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
