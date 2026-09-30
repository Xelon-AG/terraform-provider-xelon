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

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource                = (*sshKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*sshKeyResource)(nil)
	_ resource.ResourceWithImportState = (*sshKeyResource)(nil)
)

// sshKeyResource is the SSH key resource implementation.
type sshKeyResource struct {
	client *xelon.Client
}

// sshKeyResourceModel maps the SSH key resource schema data.
type sshKeyResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	PublicKey sshPublicKey `tfsdk:"public_key"`
}

func NewSSHKeyResource() resource.Resource {
	return &sshKeyResource{}
}

func (r *sshKeyResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_ssh_key"
}

func (r *sshKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The SSH key resource allows you to manage Xelon SSH keys.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the SSH key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The SSH key name.",
				Required:            true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "The public SSH key material.",
				Required:            true,
				CustomType:          sshPublicKeyType{},
			},
		},
	}
}

func (r *sshKeyResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *sshKeyResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data sshKeyResourceModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	createRequest := &xelon.SSHKeyCreateRequest{
		Name:      data.Name.ValueString(),
		PublicKey: data.PublicKey.NormalizedValue(),
	}
	tflog.Debug(ctx, "creating SSH key", map[string]any{"name": data.Name.ValueString()})

	tflog.Trace(ctx, "creating SSH key via API", map[string]any{"payload": createRequest})
	createdSSHKey, _, err := r.client.SSHKeys.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create SSH key", err.Error())
		return
	}
	tflog.Trace(ctx, "received SSH key from API", map[string]any{"data": createdSSHKey})

	tflog.Debug(ctx, "created SSH key", map[string]any{
		"id":   createdSSHKey.ID,
		"name": createdSSHKey.Name,
	})

	data.fromAPI(createdSSHKey)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *sshKeyResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data sshKeyResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	sshKeyID := data.ID.ValueString()

	tflog.Debug(ctx, "reading SSH key", map[string]any{"ssh_key_id": sshKeyID})

	tflog.Trace(ctx, "reading SSH key via API", map[string]any{"ssh_key_id": sshKeyID})
	sshKey, resp, err := r.client.SSHKeys.Get(ctx, sshKeyID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "SSH key not found; removing from state", map[string]any{"ssh_key_id": sshKeyID})
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to read SSH key", err.Error())
		return
	}
	tflog.Trace(ctx, "received SSH key from API", map[string]any{"data": sshKey})

	data.fromAPI(sshKey)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *sshKeyResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data sshKeyResourceModel

	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	sshKeyID := data.ID.ValueString()

	updateRequest := &xelon.SSHKeyUpdateRequest{
		Name:      data.Name.ValueString(),
		PublicKey: data.PublicKey.NormalizedValue(),
	}
	tflog.Debug(ctx, "updating SSH key", map[string]any{"ssh_key_id": sshKeyID, "name": data.Name.ValueString()})

	tflog.Trace(ctx, "updating SSH key via API", map[string]any{"ssh_key_id": sshKeyID, "payload": updateRequest})
	updatedSSHKey, _, err := r.client.SSHKeys.Update(ctx, sshKeyID, updateRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to update SSH key", err.Error())
		return
	}
	tflog.Trace(ctx, "received SSH key from API", map[string]any{"data": updatedSSHKey})

	tflog.Debug(ctx, "updated SSH key", map[string]any{
		"ssh_key_id": updatedSSHKey.ID,
		"name":       updatedSSHKey.Name,
	})

	data.fromAPI(updatedSSHKey)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *sshKeyResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data sshKeyResourceModel

	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	sshKeyID := data.ID.ValueString()

	tflog.Debug(ctx, "deleting SSH key", map[string]any{"ssh_key_id": sshKeyID})

	tflog.Trace(ctx, "deleting SSH key via API", map[string]any{"ssh_key_id": sshKeyID})
	resp, err := r.client.SSHKeys.Delete(ctx, sshKeyID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "SSH key already absent", map[string]any{"ssh_key_id": sshKeyID})
			return
		}
		response.Diagnostics.AddError("Unable to delete SSH key", err.Error())
		return
	}
	tflog.Debug(ctx, "deleted SSH key", map[string]any{"ssh_key_id": sshKeyID})
}

func (r *sshKeyResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func (m *sshKeyResourceModel) fromAPI(sshKey *xelon.SSHKey) {
	m.ID = types.StringValue(sshKey.ID)
	m.Name = types.StringValue(sshKey.Name)
	m.PublicKey = newSSHPublicKeyValue(sshKey.PublicKey)
}
