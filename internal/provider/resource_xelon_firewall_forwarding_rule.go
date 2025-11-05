package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	_ resource.Resource              = (*firewallForwardingRuleResource)(nil)
	_ resource.ResourceWithConfigure = (*firewallForwardingRuleResource)(nil)
)

// firewallForwardingRuleResource is the firewall forwarding rule resource implementation.
type firewallForwardingRuleResource struct {
	client *xelon.Client
}

// firewallForwardingRuleResourceModel maps the firewall forwarding rule resource schema data.
type firewallForwardingRuleResourceModel struct {
	DestinationIPAddresses types.Set    `tfsdk:"destination_ipv4_addresses"` // []types.String
	FirewallID             types.String `tfsdk:"firewall_id"`
	FromPort               types.Int64  `tfsdk:"from_port"`
	ID                     types.String `tfsdk:"id"`
	Protocol               types.String `tfsdk:"protocol"`
	SourceIPAddresses      types.Set    `tfsdk:"source_ipv4_addresses"` // []types.String
	ToPort                 types.Int64  `tfsdk:"to_port"`
	Type                   types.String `tfsdk:"type"`
}

func NewFirewallForwardingRuleResource() resource.Resource {
	return &firewallForwardingRuleResource{}
}

func (r *firewallForwardingRuleResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_firewall_forwarding_rule"
}

func (r *firewallForwardingRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The firewall forwarding rule resource allows you to manage forwarding rules for firewalls.

A forwarding rules specifies how to route network traffic to the Devices.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"destination_ipv4_addresses": schema.SetAttribute{
				MarkdownDescription: "The list of IP addresses as destination of the firewall. Must contain only one element for `inbound` rule.",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"firewall_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the firewall.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"from_port": schema.Int64Attribute{
				MarkdownDescription: "The start port: the port on the firewall side for `inbound` rule or Device port for `outbound` rule.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the forwarding rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "The protocol. Must be one of `icmp`, `tcp` or `udp`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"icmp", "tcp", "udp"}...),
				},
			},
			"source_ipv4_addresses": schema.SetAttribute{
				MarkdownDescription: "The list of IP addresses as source of the firewall. Must contain only one element for `outbound` rule.",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"to_port": schema.Int64Attribute{
				MarkdownDescription: "The end port: the port on the firewall side for `outbound` rule or Device port for `inbound` rule.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the forwarding rule. Must be one of `inbound` or `outbound`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"inbound", "outbound"}...),
				},
			},
		},
	}
}

func (r *firewallForwardingRuleResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *firewallForwardingRuleResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data firewallForwardingRuleResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueString()
	tfDestinationIPAddresses := make([]types.String, 0, len(data.DestinationIPAddresses.Elements()))
	diags = data.DestinationIPAddresses.ElementsAs(ctx, &tfDestinationIPAddresses, false)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	var destinationIPAddresses []string
	for _, tfDestinationIPAddress := range tfDestinationIPAddresses {
		destinationIPAddresses = append(destinationIPAddresses, tfDestinationIPAddress.ValueString())
	}
	tfSourceIPAddresses := make([]types.String, 0, len(data.SourceIPAddresses.Elements()))
	diags = data.SourceIPAddresses.ElementsAs(ctx, &tfSourceIPAddresses, false)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	var sourceIPAddresses []string
	for _, tfSourceIPAddress := range tfSourceIPAddresses {
		sourceIPAddresses = append(sourceIPAddresses, tfSourceIPAddress.ValueString())
	}

	ruleType := data.Type.ValueString()
	createRequest := &xelon.FirewallCreateForwardingRuleRequest{
		FirewallForwardingRule: xelon.FirewallForwardingRule{
			Protocol: data.Protocol.ValueString(),
			Type:     ruleType,
		},
	}
	if ruleType == "inbound" {
		createRequest.DestinationIPAddress = destinationIPAddresses[0]
		createRequest.SourceIPAddresses = sourceIPAddresses
		createRequest.InternalPort = (int)(data.ToPort.ValueInt64())
		createRequest.ExternalPort = (int)(data.FromPort.ValueInt64())
	}
	if ruleType == "outbound" {
		createRequest.DestinationIPAddresses = destinationIPAddresses
		createRequest.SourceIPAddress = sourceIPAddresses[0]
		createRequest.InternalPort = (int)(data.FromPort.ValueInt64())
		createRequest.ExternalPort = (int)(data.ToPort.ValueInt64())
	}

	tflog.Debug(ctx, "Creating firewall forwarding rule", map[string]any{"firewall_id": firewallID, "payload": createRequest})
	forwardingRule, _, err := r.client.Firewalls.CreateForwardingRule(ctx, firewallID, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create forwarding rule", err.Error())
		return
	}
	tflog.Debug(ctx, "Created firewall forwarding rule", map[string]any{"data": forwardingRule})

	// map response body to attributes
	data.ID = types.StringValue(forwardingRule.ID)

	// Map IP addresses from API response
	destIPAddresses := make([]string, 0, len(forwardingRule.DestinationIPAddresses))
	destIPAddresses = append(destIPAddresses, forwardingRule.DestinationIPAddresses...)
	data.DestinationIPAddresses, diags = types.SetValueFrom(ctx, types.StringType, destIPAddresses)
	response.Diagnostics.Append(diags...)

	srcIPAddresses := make([]string, 0, len(forwardingRule.SourceIPAddresses))
	srcIPAddresses = append(srcIPAddresses, forwardingRule.SourceIPAddresses...)
	data.SourceIPAddresses, diags = types.SetValueFrom(ctx, types.StringType, srcIPAddresses)
	response.Diagnostics.Append(diags...)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *firewallForwardingRuleResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data firewallForwardingRuleResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueString()
	tflog.Debug(ctx, "Getting firewall with forwarding rules", map[string]any{"firewall_id": firewallID})
	firewall, resp, err := r.client.Firewalls.Get(ctx, firewallID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the firewall (and included forwarding rules) is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get firewall with forwarding rules", err.Error())
		return
	}
	tflog.Debug(ctx, "Got firewall with forwarding rules", map[string]any{"data": firewall})

	var forwardingRule *xelon.FirewallForwardingRule
	forwardingRuleID := data.ID.ValueString()
	for _, fForwardingRule := range firewall.ForwardingRules {
		if fForwardingRule.ID == forwardingRuleID {
			forwardingRule = &fForwardingRule
			break
		}
	}
	if forwardingRule == nil {
		// if the forwarding rule is somehow already destroyed, mark as successfully gone
		response.State.RemoveResource(ctx)
		return
	}

	// map response body to attributes
	destinationIPAddresses := make([]string, 0, len(forwardingRule.DestinationIPAddresses))
	destinationIPAddresses = append(destinationIPAddresses, forwardingRule.DestinationIPAddresses...)
	data.DestinationIPAddresses, diags = types.SetValueFrom(ctx, types.StringType, destinationIPAddresses)
	response.Diagnostics.Append(diags...)
	sourceIPAddresses := make([]string, 0, len(forwardingRule.SourceIPAddresses))
	sourceIPAddresses = append(sourceIPAddresses, forwardingRule.SourceIPAddresses...)
	data.SourceIPAddresses, diags = types.SetValueFrom(ctx, types.StringType, sourceIPAddresses)
	response.Diagnostics.Append(diags...)

	if forwardingRule.Type == "inbound" {
		data.FromPort = types.Int64Value(int64(forwardingRule.ExternalPort))
		data.ToPort = types.Int64Value(int64(forwardingRule.InternalPort))
	}
	if forwardingRule.Type == "outbound" {
		data.FromPort = types.Int64Value(int64(forwardingRule.InternalPort))
		data.ToPort = types.Int64Value(int64(forwardingRule.ExternalPort))
	}

	data.FirewallID = types.StringValue(firewallID)
	data.ID = types.StringValue(forwardingRule.ID)
	data.Protocol = types.StringValue(forwardingRule.Protocol)
	data.Type = types.StringValue(forwardingRule.Type)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *firewallForwardingRuleResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data firewallForwardingRuleResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueString()
	forwardingRuleID := data.ID.ValueString()
	tfDestinationIPAddresses := make([]types.String, 0, len(data.DestinationIPAddresses.Elements()))
	diags = data.DestinationIPAddresses.ElementsAs(ctx, &tfDestinationIPAddresses, false)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	var destinationIPAddresses []string
	for _, tfDestinationIPAddress := range tfDestinationIPAddresses {
		destinationIPAddresses = append(destinationIPAddresses, tfDestinationIPAddress.ValueString())
	}
	tfSourceIPAddresses := make([]types.String, 0, len(data.SourceIPAddresses.Elements()))
	diags = data.SourceIPAddresses.ElementsAs(ctx, &tfSourceIPAddresses, false)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	var sourceIPAddresses []string
	for _, tfSourceIPAddress := range tfSourceIPAddresses {
		sourceIPAddresses = append(sourceIPAddresses, tfSourceIPAddress.ValueString())
	}

	ruleType := data.Type.ValueString()
	updateRequest := &xelon.FirewallUpdateForwardingRuleRequest{
		FirewallForwardingRule: xelon.FirewallForwardingRule{
			Protocol: data.Protocol.ValueString(),
			Type:     ruleType,
		},
	}
	if ruleType == "inbound" {
		updateRequest.DestinationIPAddress = destinationIPAddresses[0]
		updateRequest.SourceIPAddresses = sourceIPAddresses
		updateRequest.InternalPort = (int)(data.ToPort.ValueInt64())
		updateRequest.ExternalPort = (int)(data.FromPort.ValueInt64())
	}
	if ruleType == "outbound" {
		updateRequest.DestinationIPAddresses = destinationIPAddresses
		updateRequest.SourceIPAddress = sourceIPAddresses[0]
		updateRequest.InternalPort = (int)(data.FromPort.ValueInt64())
		updateRequest.ExternalPort = (int)(data.ToPort.ValueInt64())
	}

	tflog.Debug(ctx, "Updating forwarding rule", map[string]any{
		"firewall_id":        firewallID,
		"forwarding_rule_id": forwardingRuleID,
		"payload":            updateRequest,
	})
	forwardingRule, _, err := r.client.Firewalls.UpdateForwardingRule(ctx, firewallID, forwardingRuleID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update forwarding rule", err.Error())
		return
	}
	tflog.Debug(ctx, "Updated forwarding rule", map[string]any{
		"firewall_id":        firewallID,
		"forwarding_rule_id": forwardingRuleID,
		"data":               forwardingRule,
	})

	// map response body to attributes
	destinationIPAddresses = make([]string, 0, len(forwardingRule.DestinationIPAddresses))
	destinationIPAddresses = append(destinationIPAddresses, forwardingRule.DestinationIPAddresses...)
	data.DestinationIPAddresses, diags = types.SetValueFrom(ctx, types.StringType, destinationIPAddresses)
	response.Diagnostics.Append(diags...)
	sourceIPAddresses = make([]string, 0, len(forwardingRule.SourceIPAddresses))
	sourceIPAddresses = append(sourceIPAddresses, forwardingRule.SourceIPAddresses...)
	data.SourceIPAddresses, diags = types.SetValueFrom(ctx, types.StringType, sourceIPAddresses)
	response.Diagnostics.Append(diags...)

	if forwardingRule.Type == "inbound" {
		data.FromPort = types.Int64Value(int64(forwardingRule.ExternalPort))
		data.ToPort = types.Int64Value(int64(forwardingRule.InternalPort))
	}
	if forwardingRule.Type == "outbound" {
		data.FromPort = types.Int64Value(int64(forwardingRule.InternalPort))
		data.ToPort = types.Int64Value(int64(forwardingRule.ExternalPort))
	}

	data.FirewallID = types.StringValue(firewallID)
	data.ID = types.StringValue(forwardingRule.ID)
	data.Protocol = types.StringValue(forwardingRule.Protocol)
	data.Type = types.StringValue(forwardingRule.Type)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *firewallForwardingRuleResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data firewallForwardingRuleResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	firewallID := data.FirewallID.ValueString()
	forwardingRuleID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting forwarding rule", map[string]any{
		"firewall_id":        firewallID,
		"forwarding_rule_id": forwardingRuleID,
	})
	_, err := r.client.Firewalls.DeleteForwardingRule(ctx, firewallID, forwardingRuleID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete forwarding rule", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted forwarding rule", map[string]any{
		"firewall_id":        firewallID,
		"forwarding_rule_id": forwardingRuleID,
	})
}
