package provider

import (
	"context"
	"net/http"
	"strconv"

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
	_ resource.Resource              = (*loadBalancerForwardingRuleResource)(nil)
	_ resource.ResourceWithConfigure = (*loadBalancerForwardingRuleResource)(nil)
)

// loadBalancerForwardingRuleResource is the load balancer forwarding rule resource implementation.
type loadBalancerForwardingRuleResource struct {
	client *xelon.Client
}

// loadBalancerForwardingRuleResourceModel maps the load balancer forwarding rule resource schema data.
type loadBalancerForwardingRuleResourceModel struct {
	FromPort       types.Int64  `tfsdk:"from_port"`
	ID             types.String `tfsdk:"id"`
	IPAddresses    types.Set    `tfsdk:"ipv4_addresses"` // []types.String
	LoadBalancerID types.String `tfsdk:"load_balancer_id"`
	ToPort         types.Int64  `tfsdk:"to_port"`
}

func NewLoadBalancerForwardingRuleResource() resource.Resource {
	return &loadBalancerForwardingRuleResource{}
}

func (r *loadBalancerForwardingRuleResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_load_balancer_forwarding_rule"
}

func (r *loadBalancerForwardingRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The load balancer forwarding rule resource allows you to manage forwarding rules for load balancers.

A forwarding rules specifies how to route network traffic to the Devices.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"from_port": schema.Int64Attribute{
				MarkdownDescription: "The port on the load balancer side.",
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
			"ipv4_addresses": schema.SetAttribute{
				MarkdownDescription: "The list of IP addresses of the forwarding rule.",
				ElementType:         types.StringType,
				Required:            true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"load_balancer_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the load balancer.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"to_port": schema.Int64Attribute{
				MarkdownDescription: "The port on the Device side.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
		},
	}
}

func (r *loadBalancerForwardingRuleResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *loadBalancerForwardingRuleResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data loadBalancerForwardingRuleResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := data.LoadBalancerID.ValueString()
	tfIPAddresses := make([]types.String, 0, len(data.IPAddresses.Elements()))
	diags = data.IPAddresses.ElementsAs(ctx, &tfIPAddresses, false)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	var ipAddresses []string
	for _, tfIPAddress := range tfIPAddresses {
		ipAddresses = append(ipAddresses, tfIPAddress.ValueString())
	}
	createRequest := &xelon.LoadBalancerCreateForwardingRuleRequest{
		LoadBalancerForwardingRule: xelon.LoadBalancerForwardingRule{
			IPAddresses: ipAddresses,
			Ports: []int{
				int(data.FromPort.ValueInt64()),
				int(data.ToPort.ValueInt64()),
			},
		},
	}
	tflog.Debug(ctx, "Creating forwarding rule", map[string]any{"load_balancer_id": loadBalancerID, "payload": createRequest})
	forwardingRule, _, err := r.client.LoadBalancers.CreateForwardingRule(ctx, loadBalancerID, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create forwarding rule", err.Error())
		return
	}
	tflog.Debug(ctx, "Created forwarding rule", map[string]any{"data": forwardingRule})

	// map response body to attributes
	if len(forwardingRule.Ports) == 2 {
		data.FromPort = types.Int64Value(int64(forwardingRule.Ports[0]))
		data.ToPort = types.Int64Value(int64(forwardingRule.Ports[1]))
	} else {
		tflog.Warn(ctx, "Fallback to defined ports in configuration, because port count from backend API is not correct",
			map[string]any{"ports": forwardingRule.Ports},
		)
	}
	ipAddresses = make([]string, 0, len(forwardingRule.IPAddresses))
	ipAddresses = append(ipAddresses, forwardingRule.IPAddresses...)
	data.ID = types.StringValue(strconv.Itoa(forwardingRule.ID))
	data.IPAddresses, diags = types.SetValueFrom(ctx, types.StringType, ipAddresses)
	response.Diagnostics.Append(diags...)
	data.LoadBalancerID = types.StringValue(loadBalancerID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *loadBalancerForwardingRuleResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data loadBalancerForwardingRuleResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := data.LoadBalancerID.ValueString()
	tflog.Debug(ctx, "Getting load balancer with forwarding rules", map[string]any{"load_balancer_id": loadBalancerID})
	loadBalancer, resp, err := r.client.LoadBalancers.Get(ctx, loadBalancerID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the load balancer (and included forwarding rules) is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get load balancer with forwarding rules", err.Error())
		return
	}
	tflog.Debug(ctx, "Got load balancer with forwarding rules", map[string]any{"data": loadBalancer})

	var forwardingRule *xelon.LoadBalancerForwardingRule
	forwardingRuleID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to convert forwarding rule id", err.Error())
		return
	}
	for _, lbForwardingRule := range loadBalancer.ForwardingRules {
		if lbForwardingRule.ID == forwardingRuleID {
			forwardingRule = &lbForwardingRule
			break
		}
	}
	if forwardingRule == nil {
		// if the forwarding rule is somehow already destroyed, mark as successfully gone
		response.State.RemoveResource(ctx)
		return
	}

	// map response body to attributes
	if len(forwardingRule.Ports) == 2 {
		data.FromPort = types.Int64Value(int64(forwardingRule.Ports[0]))
		data.ToPort = types.Int64Value(int64(forwardingRule.Ports[1]))
	} else {
		tflog.Warn(ctx, "Fallback to defined ports in configuration, because port count from backend API is not correct",
			map[string]any{"ports": forwardingRule.Ports},
		)
	}
	ipAddresses := make([]string, 0, len(forwardingRule.IPAddresses))
	ipAddresses = append(ipAddresses, forwardingRule.IPAddresses...)
	data.ID = types.StringValue(strconv.Itoa(forwardingRule.ID))
	data.IPAddresses, diags = types.SetValueFrom(ctx, types.StringType, ipAddresses)
	response.Diagnostics.Append(diags...)
	data.LoadBalancerID = types.StringValue(loadBalancerID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *loadBalancerForwardingRuleResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data loadBalancerForwardingRuleResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := data.LoadBalancerID.ValueString()
	forwardingRuleID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to convert forwarding rule id", err.Error())
		return
	}
	tfIPAddresses := make([]types.String, 0, len(data.IPAddresses.Elements()))
	diags = data.IPAddresses.ElementsAs(ctx, &tfIPAddresses, false)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	var ipAddresses []string
	for _, tfIPAddress := range tfIPAddresses {
		ipAddresses = append(ipAddresses, tfIPAddress.ValueString())
	}
	updateRequest := &xelon.LoadBalancerUpdateForwardingRuleRequest{
		LoadBalancerForwardingRule: xelon.LoadBalancerForwardingRule{
			IPAddresses: ipAddresses,
			Ports: []int{
				int(data.FromPort.ValueInt64()),
				int(data.ToPort.ValueInt64()),
			},
		},
	}
	tflog.Debug(ctx, "Updating forwarding rule", map[string]any{
		"forwarding_rule_id": forwardingRuleID,
		"load_balancer_id":   loadBalancerID,
		"payload":            updateRequest,
	})
	forwardingRule, _, err := r.client.LoadBalancers.UpdateForwardingRule(ctx, loadBalancerID, forwardingRuleID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update forwarding rule", err.Error())
		return
	}
	tflog.Debug(ctx, "Updated forwarding rule", map[string]any{
		"forwarding_rule_id": forwardingRuleID,
		"load_balancer_id":   loadBalancerID,
		"data":               forwardingRule,
	})

	// map response body to attributes
	if len(forwardingRule.Ports) == 2 {
		data.FromPort = types.Int64Value(int64(forwardingRule.Ports[0]))
		data.ToPort = types.Int64Value(int64(forwardingRule.Ports[1]))
	} else {
		tflog.Warn(ctx, "Fallback to defined ports in configuration, because port count from backend API is not correct",
			map[string]any{"ports": forwardingRule.Ports},
		)
	}
	ipAddresses = make([]string, 0, len(forwardingRule.IPAddresses))
	ipAddresses = append(ipAddresses, forwardingRule.IPAddresses...)
	data.ID = types.StringValue(strconv.Itoa(forwardingRule.ID))
	data.IPAddresses, diags = types.SetValueFrom(ctx, types.StringType, ipAddresses)
	response.Diagnostics.Append(diags...)
	data.LoadBalancerID = types.StringValue(loadBalancerID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *loadBalancerForwardingRuleResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data loadBalancerForwardingRuleResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := data.LoadBalancerID.ValueString()
	forwardingRuleID, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Unable to convert forwarding rule id", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleting forwarding rule", map[string]any{
		"forwarding_rule_id": forwardingRuleID,
		"load_balancer_id":   loadBalancerID,
	})
	_, err = r.client.LoadBalancers.DeleteForwardingRule(ctx, loadBalancerID, forwardingRuleID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete forwarding rule", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted forwarding rule", map[string]any{
		"forwarding_rule_id": forwardingRuleID,
		"load_balancer_id":   loadBalancerID,
	})
}
