package provider

import (
	"context"
	"fmt"

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

SSH keys can be injected into devices during creation for secure authentication.
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
				MarkdownDescription: "The public SSH key content.",
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

	keyID := data.ID.ValueString()
	keyName := data.Name.ValueString()
	if keyID == "" && keyName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	var sshKey *xelon.SSHKey
	var err error

	// If ID is provided, get directly by ID
	if keyID != "" {
		tflog.Debug(ctx, "Getting SSH key by ID", map[string]any{"ssh_key_id": keyID})
		sshKey, _, err = d.client.SSHKeys.Get(ctx, keyID)
		if err != nil {
			response.Diagnostics.AddError(
				"Unable to get SSH key by ID",
				fmt.Sprintf("Error: %s", err.Error()),
			)
			return
		}

		// If name is also provided, validate it matches
		if keyName != "" && keyName != sshKey.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual SSH key name are different: expected '%s', got '%s'.", keyName, sshKey.Name),
			)
			return
		}
	} else {
		// Search by name using server-side filtering
		opts := &xelon.SSHKeyListOptions{
			Sort:   "name",
			Search: keyName,
		}

		tflog.Debug(ctx, "Getting SSH keys with server-side search", map[string]any{"search": keyName})
		sshKeys, _, err := d.client.SSHKeys.List(ctx, opts)
		if err != nil {
			response.Diagnostics.AddError("Unable to list SSH keys", err.Error())
			return
		}
		tflog.Debug(ctx, "Got filtered SSH keys from API", map[string]any{"count": len(sshKeys)})

		// API search may return partial matches, find exact match
		for _, key := range sshKeys {
			if keyName == key.Name {
				sshKey = &key
				break
			}
		}

		if sshKey == nil {
			response.Diagnostics.AddError(
				"No search results",
				"No SSH key found matching the specified criteria. Please refine your search.",
			)
			return
		}
	}

	// map response body to attributes
	data.ID = types.StringValue(sshKey.ID)
	data.Name = types.StringValue(sshKey.Name)
	data.PublicKey = types.StringValue(sshKey.PublicKey)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
