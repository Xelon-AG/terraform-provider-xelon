package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*dnsRecordResource)(nil)
	_ resource.ResourceWithConfigure   = (*dnsRecordResource)(nil)
	_ resource.ResourceWithImportState = (*dnsRecordResource)(nil)
)

// dnsRecordResource is the dns record resource implementation.
type dnsRecordResource struct {
	client *xelon.Client
}

// dnsRecordResourceModel maps the dns record resource schema data.
type dnsRecordResourceModel struct {
	Content  types.String `tfsdk:"content"`
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	RecordID types.Int64  `tfsdk:"record_id"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Type     types.String `tfsdk:"type"`
	ZoneID   types.String `tfsdk:"zone_id"`
}

func NewDNSRecordResource() resource.Resource {
	return &dnsRecordResource{}
}

func (r *dnsRecordResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_dns_record"
}

func (r *dnsRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The DNS record resource allows you to manage flat DNS records in a Xelon DNS zone.

Supported record types in v0: A, AAAA, CNAME, TXT, NS, ALIAS, PTR.
Record types that require additional structured fields, such as MX, SRV, CAA, RP, SSHFP, and TLSA, are not supported yet.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"content": schema.StringAttribute{
				MarkdownDescription: "DNS record content/value, such as an IP address, hostname, or TXT value.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of the DNS record in the format `<zone_id>/<record_id>`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: `DNS record name, such as "www", "@", or "_sip._tcp".`,
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"record_id": schema.Int64Attribute{
				MarkdownDescription: "Backend DNS record ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"ttl": schema.Int64Attribute{
				MarkdownDescription: "DNS record TTL in seconds.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "DNS record type. Supported v0 types are `A`, `AAAA`, `CNAME`, `TXT`, `NS`, `ALIAS`, and `PTR`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(supportedV0DNSRecordTypes()...),
				},
			},
			"zone_id": schema.StringAttribute{
				MarkdownDescription: "ID of the DNS zone owning this record.",
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

func (r *dnsRecordResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *dnsRecordResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data dnsRecordResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneID := data.ZoneID.ValueString()
	createRequest := &xelon.DNSRecordCreateRequest{
		Host:   data.Name.ValueString(),
		Record: data.Content.ValueString(),
		TTL:    int(data.TTL.ValueInt64()),
		Type:   xelon.DNSRecordType(data.Type.ValueString()),
	}

	tflog.Debug(ctx, "creating DNS record", map[string]any{
		"zone_id": zoneID,
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
	})

	tflog.Trace(ctx, "creating DNS record via API", map[string]any{
		"zone_id": zoneID,
		"name":    data.Name.ValueString(),
		"type":    data.Type.ValueString(),
	})
	_, err := r.client.Domains.CreateRecord(ctx, zoneID, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create DNS record", err.Error())
		return
	}

	tflog.Trace(ctx, "listing DNS records via API (created record lookup)", map[string]any{"zone_id": zoneID})
	records, resp, err := r.client.Domains.ListRecords(ctx, zoneID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			response.Diagnostics.AddError(
				"Unable to read DNS record",
				"DNS record was created, but the parent DNS zone could not be found while refreshing state.",
			)
			return
		}
		response.Diagnostics.AddError("Unable to read DNS record", err.Error())
		return
	}

	record, matchCount := findCreatedDNSRecord(records, &data)
	switch matchCount {
	case 0:
		response.Diagnostics.AddError(
			"Unable to find created DNS record",
			"DNS record was created, but the provider could not find it in the zone record list.",
		)
		return
	case 1:
		data.fromAPI(record, zoneID)
		tflog.Debug(ctx, "created DNS record", map[string]any{
			"zone_id":   zoneID,
			"record_id": strconv.FormatInt(int64(record.ID), 10),
		})
	default:
		response.Diagnostics.AddError(
			"Unable to identify created DNS record",
			"DNS record was created, but multiple records matched the requested values. The provider cannot determine which backend record ID belongs to this Terraform resource.",
		)
		return
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *dnsRecordResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data dnsRecordResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneID, recordID, err := parseDNSRecordCompositeID(data.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Invalid DNS record ID", err.Error())
		return
	}

	tflog.Debug(ctx, "reading DNS record", map[string]any{
		"zone_id":   zoneID,
		"record_id": strconv.FormatInt(recordID, 10),
	})

	tflog.Trace(ctx, "listing DNS records via API (record lookup)", map[string]any{"zone_id": zoneID})
	records, resp, err := r.client.Domains.ListRecords(ctx, zoneID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read DNS record", err.Error())
		return
	}

	record := findDNSRecordByID(records, recordID)
	if record == nil {
		response.State.RemoveResource(ctx)
		return
	}

	data.fromAPI(record, zoneID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *dnsRecordResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data dnsRecordResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneID, recordID, err := parseDNSRecordCompositeID(data.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Invalid DNS record ID", err.Error())
		return
	}

	updateRequest := &xelon.DNSRecordUpdateRequest{
		Host:   data.Name.ValueString(),
		Record: data.Content.ValueString(),
		TTL:    int(data.TTL.ValueInt64()),
		Type:   xelon.DNSRecordType(data.Type.ValueString()),
	}

	tflog.Debug(ctx, "updating DNS record", map[string]any{
		"zone_id":   zoneID,
		"record_id": strconv.FormatInt(recordID, 10),
		"name":      data.Name.ValueString(),
		"type":      data.Type.ValueString(),
	})

	tflog.Trace(ctx, "updating DNS record via API", map[string]any{
		"zone_id":   zoneID,
		"record_id": strconv.FormatInt(recordID, 10),
		"name":      data.Name.ValueString(),
		"type":      data.Type.ValueString(),
	})
	_, err = r.client.Domains.UpdateRecord(ctx, zoneID, int(recordID), updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update DNS record", err.Error())
		return
	}

	tflog.Trace(ctx, "listing DNS records via API (updated record lookup)", map[string]any{"zone_id": zoneID})
	records, resp, err := r.client.Domains.ListRecords(ctx, zoneID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			response.Diagnostics.AddError(
				"Unable to read DNS record",
				"DNS record was updated, but the parent DNS zone could not be found while refreshing state.",
			)
			return
		}
		response.Diagnostics.AddError("Unable to read DNS record", err.Error())
		return
	}

	record := findDNSRecordByID(records, recordID)
	if record == nil {
		response.Diagnostics.AddError(
			"Unable to read DNS record",
			"DNS record was updated, but the provider could not find it in the zone record list.",
		)
		return
	}

	data.fromAPI(record, zoneID)

	tflog.Debug(ctx, "updated DNS record", map[string]any{
		"zone_id":   zoneID,
		"record_id": strconv.FormatInt(recordID, 10),
	})

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *dnsRecordResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data dnsRecordResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	zoneID, recordID, err := parseDNSRecordCompositeID(data.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Invalid DNS record ID", err.Error())
		return
	}

	tflog.Debug(ctx, "deleting DNS record", map[string]any{
		"zone_id":   zoneID,
		"record_id": strconv.FormatInt(recordID, 10),
	})

	tflog.Trace(ctx, "deleting DNS record via API", map[string]any{
		"zone_id":   zoneID,
		"record_id": strconv.FormatInt(recordID, 10),
	})
	resp, err := r.client.Domains.DeleteRecord(ctx, zoneID, int(recordID))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return
		}
		response.Diagnostics.AddError("Unable to delete DNS record", err.Error())
		return
	}

	tflog.Debug(ctx, "deleted DNS record", map[string]any{
		"zone_id":   zoneID,
		"record_id": strconv.FormatInt(recordID, 10),
	})
}

func (r *dnsRecordResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	zoneID, recordID, err := parseDNSRecordCompositeID(request.ID)
	if err != nil {
		response.Diagnostics.AddError("Invalid import identifier", err.Error())
		return
	}

	tflog.Debug(ctx, "importing DNS record", map[string]any{
		"zone_id":   zoneID,
		"record_id": strconv.FormatInt(recordID, 10),
	})

	tflog.Trace(ctx, "listing DNS records via API (import record lookup)", map[string]any{"zone_id": zoneID})
	records, resp, err := r.client.Domains.ListRecords(ctx, zoneID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			response.Diagnostics.AddError("Unable to import DNS record", "The parent DNS zone was not found.")
			return
		}
		response.Diagnostics.AddError("Unable to import DNS record", err.Error())
		return
	}

	record := findDNSRecordByID(records, recordID)
	if record == nil {
		response.Diagnostics.AddError("Unable to import DNS record", "No DNS record with the given backend record ID was found in the parent DNS zone.")
		return
	}

	var data dnsRecordResourceModel
	data.fromAPI(record, zoneID)

	diags := response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (m *dnsRecordResourceModel) fromAPI(record *xelon.DNSRecord, zoneID string) {
	recordID := int64(record.ID)

	m.Content = types.StringValue(record.Record)
	m.ID = types.StringValue(buildDNSRecordCompositeID(zoneID, recordID))
	m.Name = types.StringValue(record.Host)
	m.RecordID = types.Int64Value(recordID)
	m.TTL = types.Int64Value(int64(record.TTL))
	m.Type = types.StringValue(string(record.Type))
	m.ZoneID = types.StringValue(zoneID)
}

func buildDNSRecordCompositeID(zoneID string, recordID int64) string {
	return fmt.Sprintf("%s/%d", zoneID, recordID)
}

func parseDNSRecordCompositeID(id string) (string, int64, error) {
	zoneID, recordIDRaw, ok := strings.Cut(id, "/")
	if !ok || zoneID == "" || recordIDRaw == "" {
		return "", 0, fmt.Errorf("expected format: <zone_id>/<record_id>")
	}

	recordID, err := strconv.ParseInt(recordIDRaw, 10, 64)
	if err != nil || recordID <= 0 {
		return "", 0, fmt.Errorf("expected format: <zone_id>/<record_id> with a positive integer record_id")
	}

	return zoneID, recordID, nil
}

func findDNSRecordByID(records []xelon.DNSRecord, recordID int64) *xelon.DNSRecord {
	for i := range records {
		if int64(records[i].ID) == recordID {
			return &records[i]
		}
	}

	return nil
}

func findCreatedDNSRecord(records []xelon.DNSRecord, data *dnsRecordResourceModel) (*xelon.DNSRecord, int) {
	var match *xelon.DNSRecord
	matchCount := 0

	for i := range records {
		record := &records[i]
		if record.Host != data.Name.ValueString() {
			continue
		}
		if record.Type != xelon.DNSRecordType(data.Type.ValueString()) {
			continue
		}
		if record.Record != data.Content.ValueString() {
			continue
		}
		if int64(record.TTL) != data.TTL.ValueInt64() {
			continue
		}

		match = record
		matchCount++
	}

	return match, matchCount
}

func supportedV0DNSRecordTypes() []string {
	return []string{
		string(xelon.DNSRecordTypeA),
		string(xelon.DNSRecordTypeAAAA),
		string(xelon.DNSRecordTypeALIAS),
		string(xelon.DNSRecordTypeCNAME),
		string(xelon.DNSRecordTypeNS),
		string(xelon.DNSRecordTypePTR),
		string(xelon.DNSRecordTypeTXT),
	}
}
