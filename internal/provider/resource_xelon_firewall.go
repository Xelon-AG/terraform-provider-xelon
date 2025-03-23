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
	_ resource.Resource                = (*firewallResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallResource)(nil)
	_ resource.ResourceWithImportState = (*firewallResource)(nil)
)

// firewallResource is the firewall resource implementation.
type firewallResource struct {
	client *xelon.Client
}

// firewallResourceModel maps the firewall resource schema data.
type firewallResourceModel struct {
	CloudID           types.String `tfsdk:"cloud_id"`
	ExternalIPAddress types.String `tfsdk:"external_ipv4_address"`
	ID                types.String `tfsdk:"id"`
	InternalIPAddress types.String `tfsdk:"internal_ipv4_address"`
	Name              types.String `tfsdk:"name"`
	NetworkID         types.String `tfsdk:"network_id"`
	TenantID          types.String `tfsdk:"tenant_id"`
}

func NewFirewallResource() resource.Resource {
	return &firewallResource{}
}

func (r *firewallResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_firewall"
}

func (r *firewallResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The firewall resource allows you to manage Xelon firewalls.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud associated with the firewall.",
				Required:            true,
			},
			"external_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The external IP address of the firewall.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the firewall.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"internal_ipv4_address": schema.StringAttribute{
				MarkdownDescription: "The internal IP address of the firewall. If not provided, an internal IP will be automatically assigned.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The firewall name.",
				Required:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The network ID used to create the firewall.",
				Required:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the firewall belongs.",
				Required:            true,
			},
		},
	}
}

func (r *firewallResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *firewallResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data firewallResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &xelon.FirewallCreateRequest{
		CloudID:           data.CloudID.ValueString(),
		InternalNetworkID: data.NetworkID.ValueString(),
		Name:              data.Name.ValueString(),
		TenantID:          data.TenantID.ValueString(),
	}
	if data.InternalIPAddress.ValueString() != "" {
		createRequest.InternalIPAddress = data.InternalIPAddress.ValueString()
	}
	tflog.Debug(ctx, "Creating firewall", map[string]any{"payload": createRequest})
	createdFirewall, _, err := r.client.Firewalls.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create firewall", err.Error())
		return
	}
	tflog.Debug(ctx, "Created firewall", map[string]any{"data": createdFirewall})

	firewallID := createdFirewall.ID

	tflog.Info(ctx, "Waiting for firewall to be ready", map[string]any{"firewall_id": firewallID})
	err = helper.WaitFirewallStateReady(ctx, r.client, firewallID)
	if err != nil {
		response.Diagnostics.AddError("Unable to wait for firewall to be ready", err.Error())
		return
	}
	tflog.Info(ctx, "firewall is ready", map[string]any{"firewall_id": firewallID})

	tflog.Debug(ctx, "Getting firewall with enriched properties", map[string]any{"firewall_id": firewallID})
	firewall, _, err := r.client.Firewalls.Get(ctx, firewallID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get firewall", err.Error())
		return
	}
	tflog.Debug(ctx, "Got firewall with enriched properties", map[string]any{"data": firewall})

	// map response body to attributes
	data.CloudID = types.StringValue(firewall.Cloud.ID)
	data.ExternalIPAddress = types.StringValue(firewall.ExternalIPAddress)
	data.ID = types.StringValue(firewall.ID)
	data.InternalIPAddress = types.StringValue(firewall.InternalIPAddress)
	data.Name = types.StringValue(firewall.Name)
	data.TenantID = types.StringValue(firewall.Tenant.ID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *firewallResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data firewallResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting firewall", map[string]any{"firewall_id": firewallID})
	firewall, resp, err := r.client.Firewalls.Get(ctx, firewallID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the firewall is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get firewall", err.Error())
		return
	}
	tflog.Debug(ctx, "Got firewall", map[string]any{"data": firewall})

	// map response body to attributes
	data.CloudID = types.StringValue(firewall.Cloud.ID)
	data.ExternalIPAddress = types.StringValue(firewall.ExternalIPAddress)
	data.ID = types.StringValue(firewall.ID)
	data.InternalIPAddress = types.StringValue(firewall.InternalIPAddress)
	data.Name = types.StringValue(firewall.Name)
	data.TenantID = types.StringValue(firewall.Tenant.ID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *firewallResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state firewallResourceModel

	// read plan and state data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallID := state.ID.ValueString()

	if !plan.Name.Equal(state.Name) {
		updateRequest := &xelon.FirewallUpdateRequest{
			Name: plan.Name.ValueString(),
		}
		tflog.Debug(ctx, "Updating firewall", map[string]any{"firewall_id": firewallID, "payload": updateRequest})
		firewall, _, err := r.client.Firewalls.Update(ctx, firewallID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update firewall", err.Error())
			return
		}
		tflog.Debug(ctx, "Updated firewall", map[string]any{"firewall_id": firewallID, "data": firewall})

		tflog.Debug(ctx, "Getting firewall with enriched data", map[string]any{"firewall_id": firewallID})
		firewall, _, err = r.client.Firewalls.Get(ctx, firewallID)
		if err != nil {
			response.Diagnostics.AddError("Unable to get firewall", err.Error())
			return
		}
		tflog.Debug(ctx, "Got firewall with enriched data", map[string]any{"data": firewall})

		plan.Name = types.StringValue(firewall.Name)
	}

	diags := response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}

func (r *firewallResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data firewallResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting firewall", map[string]any{"firewall_id": firewallID})
	_, err := r.client.Firewalls.Delete(ctx, firewallID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete firewall", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted firewall", map[string]any{"firewall_id": firewallID})
}

func (r *firewallResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
