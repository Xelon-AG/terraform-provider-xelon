package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ datasource.DataSource              = (*sshKeyDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*sshKeyDataSource)(nil)
)

// sshKeyDataSource is the SSH key datasource implementation.
type sshKeyDataSource struct {
	client *xelon.Client
}

// sshKeyDataSourceModel maps the SSH key datasource schema data.
type sshKeyDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	PublicKey types.String `tfsdk:"public_key"`
}

func NewSSHKeyDataSource() datasource.DataSource {
	return &sshKeyDataSource{}
}

func (d *sshKeyDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_ssh_key"
}

func (d *sshKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The SSH key data source provides information about an existing SSH key.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the SSH key.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The SSH key name.",
				Computed:            true,
				Optional:            true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "The public SSH key material.",
				Computed:            true,
			},
		},
	}
}

func (d *sshKeyDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *sshKeyDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data sshKeyDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	sshKeyID := data.ID.ValueString()
	sshKeyName := data.Name.ValueString()
	if sshKeyID == "" && sshKeyName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	if sshKeyID != "" {
		tflog.Info(ctx, "Searching for SSH key by ID", map[string]any{"ssh_key_id": sshKeyID})

		tflog.Debug(ctx, "Getting SSH key", map[string]any{"ssh_key_id": sshKeyID})
		sshKey, resp, err := d.client.SSHKeys.Get(ctx, sshKeyID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
			response.Diagnostics.AddError("Unable to get SSH key", err.Error())
			return
		}
		tflog.Debug(ctx, "Got SSH key", map[string]any{"data": sshKey})

		// if name is defined check that it's equal
		if sshKeyName != "" && sshKeyName != sshKey.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual SSH key name are different: expected '%s', got '%s'.", sshKeyName, sshKey.Name),
			)
			return
		}

		// map response body to attributes
		data.ID = types.StringValue(sshKey.ID)
		data.Name = types.StringValue(sshKey.Name)
		data.PublicKey = types.StringValue(sshKey.PublicKey)
	} else {
		tflog.Info(ctx, "Searching for SSH key by name", map[string]any{"ssh_key_name": sshKeyName})

		tflog.Debug(ctx, "Getting SSH keys", map[string]any{"ssh_key_name": sshKeyName})
		sshKeys, _, err := d.client.SSHKeys.List(ctx, &xelon.SSHKeyListOptions{Search: sshKeyName})
		if err != nil {
			response.Diagnostics.AddError("Unable to search SSH keys by name", err.Error())
			return
		}
		tflog.Debug(ctx, "Got SSH keys", map[string]any{"data": sshKeys})

		if len(sshKeys) == 0 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
		if len(sshKeys) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v SSH keys.", len(sshKeys)),
			)
			return
		}

		sshKey := sshKeys[0]
		// map response body to attributes
		data.ID = types.StringValue(sshKey.ID)
		data.Name = types.StringValue(sshKey.Name)
		data.PublicKey = types.StringValue(sshKey.PublicKey)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
