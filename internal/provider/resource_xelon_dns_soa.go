package provider

import (
	"context"
	"net/http"

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
	_ resource.Resource                = (*dnsSOAResource)(nil)
	_ resource.ResourceWithConfigure   = (*dnsSOAResource)(nil)
	_ resource.ResourceWithImportState = (*dnsSOAResource)(nil)
)

// dnsSOAResource is the dns SOA settings resource implementation.
type dnsSOAResource struct {
	client *xelon.Client
}

// dnsSOAResourceModel maps the dns SOA settings resource schema data.
type dnsSOAResourceModel struct {
	AdminEmail        types.String `tfsdk:"admin_email"`
	Expire            types.Int64  `tfsdk:"expire"`
	ID                types.String `tfsdk:"id"`
	PrimaryNameserver types.String `tfsdk:"primary_nameserver"`
	Refresh           types.Int64  `tfsdk:"refresh"`
	Retry             types.Int64  `tfsdk:"retry"`
	TTL               types.Int64  `tfsdk:"ttl"`
	ZoneID            types.String `tfsdk:"zone_id"`
}

func NewDNSSOAResource() resource.Resource {
	return &dnsSOAResource{}
}

func (r *dnsSOAResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_dns_soa"
}

func (r *dnsSOAResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The DNS SOA resource manages Start of Authority settings for a Xelon DNS zone.

SOA settings are created automatically with the DNS zone. This resource manages those existing zone-owned settings by updating the complete SOA configuration. Xelon does not expose a separate SOA create or delete operation.

Deleting this Terraform resource removes SOA management from Terraform state only. It does not delete, reset, clear, or otherwise modify the remote SOA settings, and it does not delete the DNS zone.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Terraform resource ID for the DNS SOA settings. This is the owning DNS zone ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "ID of the DNS zone owning these SOA settings.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"primary_nameserver": schema.StringAttribute{
				MarkdownDescription: "Primary nameserver for the DNS zone.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"admin_email": schema.StringAttribute{
				MarkdownDescription: "Administrative email address for the DNS zone.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"refresh": schema.Int64Attribute{
				MarkdownDescription: "SOA refresh interval in seconds.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"retry": schema.Int64Attribute{
				MarkdownDescription: "SOA retry interval in seconds.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"expire": schema.Int64Attribute{
				MarkdownDescription: "SOA expire interval in seconds.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "SOA TTL in seconds.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
		},
	}
}

func (r *dnsSOAResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *dnsSOAResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data dnsSOAResourceModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()
	updateRequest := &xelon.DNSSOAUpdateRequest{
		AdminEmail: data.AdminEmail.ValueString(),
		Expire:     int(data.Expire.ValueInt64()),
		PrimaryNS:  data.PrimaryNameserver.ValueString(),
		Refresh:    int(data.Refresh.ValueInt64()),
		Retry:      int(data.Retry.ValueInt64()),
		TTL:        int(data.TTL.ValueInt64()),
	}

	tflog.Debug(ctx, "configuring DNS SOA settings", map[string]any{"zone_id": zoneID})

	tflog.Trace(ctx, "updating DNS SOA settings via API", map[string]any{"zone_id": zoneID})
	_, err := r.client.Domains.UpdateSOA(ctx, zoneID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to configure DNS SOA settings", err.Error())
		return
	}

	tflog.Debug(ctx, "configured DNS SOA settings", map[string]any{"zone_id": zoneID})

	tflog.Debug(ctx, "refreshing DNS SOA settings state after configure", map[string]any{"zone_id": zoneID})

	tflog.Trace(ctx, "reading DNS SOA settings via API (configured SOA refresh)", map[string]any{"zone_id": zoneID})
	soa, resp, err := r.client.Domains.GetSOA(ctx, zoneID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			response.Diagnostics.AddError(
				"Unable to read DNS SOA settings",
				"DNS SOA settings were configured, but the DNS zone could not be found while refreshing state.",
			)
			return
		}
		response.Diagnostics.AddError("Unable to read DNS SOA settings", err.Error())
		return
	}
	tflog.Trace(ctx, "received DNS SOA settings from API", map[string]any{
		"zone_id":            zoneID,
		"primary_nameserver": soa.PrimaryNS,
		"refresh":            soa.Refresh,
		"retry":              soa.Retry,
		"expire":             soa.Expire,
		"ttl":                soa.TTL,
	})

	data.fromAPI(soa, zoneID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *dnsSOAResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data dnsSOAResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()

	tflog.Debug(ctx, "reading DNS SOA settings", map[string]any{"zone_id": zoneID})

	tflog.Trace(ctx, "reading DNS SOA settings via API", map[string]any{"zone_id": zoneID})
	soa, resp, err := r.client.Domains.GetSOA(ctx, zoneID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read DNS SOA settings", err.Error())
		return
	}
	tflog.Trace(ctx, "received DNS SOA settings from API", map[string]any{
		"zone_id":            zoneID,
		"primary_nameserver": soa.PrimaryNS,
		"refresh":            soa.Refresh,
		"retry":              soa.Retry,
		"expire":             soa.Expire,
		"ttl":                soa.TTL,
	})

	data.fromAPI(soa, zoneID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *dnsSOAResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data dnsSOAResourceModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()
	updateRequest := &xelon.DNSSOAUpdateRequest{
		AdminEmail: data.AdminEmail.ValueString(),
		Expire:     int(data.Expire.ValueInt64()),
		PrimaryNS:  data.PrimaryNameserver.ValueString(),
		Refresh:    int(data.Refresh.ValueInt64()),
		Retry:      int(data.Retry.ValueInt64()),
		TTL:        int(data.TTL.ValueInt64()),
	}

	tflog.Debug(ctx, "updating DNS SOA settings", map[string]any{"zone_id": zoneID})

	tflog.Trace(ctx, "updating DNS SOA settings via API", map[string]any{"zone_id": zoneID})
	_, err := r.client.Domains.UpdateSOA(ctx, zoneID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update DNS SOA settings", err.Error())
		return
	}

	tflog.Debug(ctx, "updated DNS SOA settings", map[string]any{"zone_id": zoneID})

	tflog.Debug(ctx, "refreshing DNS SOA settings state after update", map[string]any{"zone_id": zoneID})

	tflog.Trace(ctx, "reading DNS SOA settings via API (updated SOA refresh)", map[string]any{"zone_id": zoneID})
	soa, resp, err := r.client.Domains.GetSOA(ctx, zoneID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			response.Diagnostics.AddError(
				"Unable to read DNS SOA settings",
				"DNS SOA settings were updated, but the DNS zone could not be found while refreshing state.",
			)
			return
		}
		response.Diagnostics.AddError("Unable to read DNS SOA settings", err.Error())
		return
	}
	tflog.Trace(ctx, "received DNS SOA settings from API", map[string]any{
		"zone_id":            zoneID,
		"primary_nameserver": soa.PrimaryNS,
		"refresh":            soa.Refresh,
		"retry":              soa.Retry,
		"expire":             soa.Expire,
		"ttl":                soa.TTL,
	})

	data.fromAPI(soa, zoneID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *dnsSOAResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data dnsSOAResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()

	tflog.Debug(ctx, "removing DNS SOA settings from Terraform state", map[string]any{
		"state_only": true,
		"zone_id":    zoneID,
	})

	tflog.Debug(ctx, "removed DNS SOA settings from Terraform state", map[string]any{
		"state_only": true,
		"zone_id":    zoneID,
	})
}

func (r *dnsSOAResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), request.ID)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("zone_id"), request.ID)...)
}

func (m *dnsSOAResourceModel) fromAPI(soa *xelon.DNSSOA, zoneID string) {
	m.AdminEmail = types.StringValue(soa.AdminEmail)
	m.Expire = types.Int64Value(int64(soa.Expire))
	m.ID = types.StringValue(zoneID)
	m.PrimaryNameserver = types.StringValue(soa.PrimaryNS)
	m.Refresh = types.Int64Value(int64(soa.Refresh))
	m.Retry = types.Int64Value(int64(soa.Retry))
	m.TTL = types.Int64Value(int64(soa.TTL))
	m.ZoneID = types.StringValue(zoneID)
}
