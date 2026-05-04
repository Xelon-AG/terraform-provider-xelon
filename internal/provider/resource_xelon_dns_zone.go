package provider

import (
	"context"
	"net/http"

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
	_ resource.Resource                = (*dnsZoneResource)(nil)
	_ resource.ResourceWithConfigure   = (*dnsZoneResource)(nil)
	_ resource.ResourceWithImportState = (*dnsZoneResource)(nil)
)

// dnsZoneResource is the dns zone resource implementation.
type dnsZoneResource struct {
	client *xelon.Client
}

// dnsZoneResourceModel maps the dns zone resource schema data.
type dnsZoneResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func NewDNSZoneResource() resource.Resource {
	return &dnsZoneResource{}
}

func (r *dnsZoneResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_dns_zone"
}

func (r *dnsZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The DNS zone resource allows you to manage DNS zones in Xelon.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the DNS zone.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the DNS zone.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *dnsZoneResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *dnsZoneResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data dnsZoneResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &xelon.DNSZoneCreateRequest{
		Domain: data.Name.ValueString(),
	}
	tflog.Debug(ctx, "Creating dns zone", map[string]any{"payload": createRequest})
	createDNSZone, _, err := r.client.Domains.CreateDNSZone(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create dns zone", err.Error())
		return
	}
	tflog.Debug(ctx, "Created dns zone", map[string]any{"data": createDNSZone})

	// map response body to attributes
	data.ID = types.StringValue(createDNSZone.ID)
	data.Name = types.StringValue(createDNSZone.Name)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *dnsZoneResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data dnsZoneResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	dnsZoneID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting dns zone", map[string]any{"dns_zone_id": dnsZoneID})
	dnsZone, resp, err := r.client.Domains.GetDNSZone(ctx, dnsZoneID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the dns zone is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get dns zone", err.Error())
		return
	}
	tflog.Debug(ctx, "Got dns zone", map[string]any{"data": dnsZone})

	// map response body to attributes
	data.ID = types.StringValue(dnsZone.ID)
	data.Name = types.StringValue(dnsZone.Name)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *dnsZoneResource) Update(ctx context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating of dns zone is not supported, resource will be re-created")
}

func (r *dnsZoneResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data dnsZoneResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	dnsZoneID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting dns zone", map[string]any{"dns_zone_id": dnsZoneID})
	_, err := r.client.Domains.DeleteDNSZone(ctx, dnsZoneID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete dns zone", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted dns zone", map[string]any{"dns_zone_id": dnsZoneID})
}

func (r *dnsZoneResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
