package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Network      types.String `tfsdk:"network"`
	SubnetSize   types.Int64  `tfsdk:"subnet_size"`
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

	createRequest := &xelon.NetworkCreateRequest{
		CloudID: data.CloudID.ValueString(),
	}
	tflog.Debug(ctx, "Creating network", map[string]any{"payload": createRequest})
	network, _, err := r.client.Networks.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create network", err.Error())
		return
	}
	tflog.Debug(ctx, "Created network", map[string]any{"data": network})

	// map response body to attributes
	data.ID = types.StringValue(network.ID)
	data.Name = types.StringValue(network.Name)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *networkResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	// TODO implement me
	panic("implement me")
}

func (r *networkResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	// TODO implement me
	panic("implement me")
}

func (r *networkResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	// TODO implement me
	panic("implement me")
}

func (r *networkResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
