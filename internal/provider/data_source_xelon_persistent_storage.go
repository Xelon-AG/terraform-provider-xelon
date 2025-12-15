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
	_ datasource.DataSource              = (*persistentStorageDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*persistentStorageDataSource)(nil)
)

// persistentStorageDataSource is the persistent storage data source implementation.
type persistentStorageDataSource struct {
	client *xelon.Client
}

// persistentStorageDataSourceModel maps the persistent storage datasource schema data.
type persistentStorageDataSourceModel struct {
	CloudID  types.String `tfsdk:"cloud_id"`
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Size     types.Int64  `tfsdk:"size"`
	TenantID types.String `tfsdk:"tenant_id"`
	UUID     types.String `tfsdk:"uuid"`
}

func NewPersistentStorageDataSource() datasource.DataSource {
	return &persistentStorageDataSource{}
}

func (d *persistentStorageDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_persistent_storage"
}

func (d *persistentStorageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The persistent storage data source provides information about an existing storages.
`,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the persistent storage.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The persistent storage name.",
				Computed:            true,
				Optional:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "The size of the persistent storage in GB.",
				Computed:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID to whom the persistent storage belongs.",
				Computed:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the persistent storage device.",
				Computed:            true,
			},
		},
	}
}

func (d *persistentStorageDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *persistentStorageDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data persistentStorageDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	persistentStorageID := data.ID.ValueString()
	persistentStorageName := data.Name.ValueString()
	if persistentStorageID == "" && persistentStorageName == "" {
		response.Diagnostics.AddError(
			"Missing required attributes",
			`The attribute "id" or "name" must be defined.`,
		)
		return
	}

	if persistentStorageID != "" {
		tflog.Info(ctx, "Searching for persistent storage by ID", map[string]any{"persistent_storage_id": persistentStorageID})

		tflog.Debug(ctx, "Getting persistent storage", map[string]any{"persistent_storage_id": persistentStorageID})
		persistentStorage, resp, err := d.client.PersistentStorages.Get(ctx, persistentStorageID)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				response.Diagnostics.AddError("No search results", "Please refine your search.")
				return
			}
			response.Diagnostics.AddError("Unable to get persistent storage", err.Error())
			return
		}
		tflog.Debug(ctx, "Got persistent storage", map[string]any{"data": persistentStorage, "persistent_storage_id": persistentStorageID})

		// if name is defined check that it's equal
		if persistentStorageName != "" && persistentStorageName != persistentStorage.Name {
			response.Diagnostics.AddError(
				"Ambiguous search result",
				fmt.Sprintf("Specified and actual persistent storage name are different: expected '%s', got '%s'.", persistentStorageName, persistentStorage.Name),
			)
			return
		}

		// map response body to attributes
		if persistentStorage.Cloud != nil {
			data.CloudID = types.StringValue(persistentStorage.Cloud.ID)
		}
		data.ID = types.StringValue(persistentStorage.ID)
		data.Name = types.StringValue(persistentStorage.Name)
		data.Size = types.Int64Value(int64(persistentStorage.Capacity))
		if persistentStorage.Tenant != nil {
			data.TenantID = types.StringValue(persistentStorage.Tenant.ID)
		}
		data.UUID = types.StringValue(persistentStorage.UUID)
	} else {
		tflog.Info(ctx, "Searching for persistent storage by name", map[string]any{"persistent_storage_name": persistentStorageName})

		tflog.Debug(ctx, "Getting persistent storages", map[string]any{"persistent_storage_name": persistentStorageName})
		persistentStorages, _, err := d.client.PersistentStorages.List(ctx, &xelon.PersistentStorageListOptions{Search: persistentStorageName})
		if err != nil {
			response.Diagnostics.AddError("Unable to search persistent storage by name", err.Error())
			return
		}
		tflog.Debug(ctx, "Got persistent storages", map[string]any{"data": persistentStorages})

		if len(persistentStorages) == 0 {
			response.Diagnostics.AddError("No search results", "Please refine your search.")
			return
		}
		if len(persistentStorages) > 1 {
			response.Diagnostics.AddError(
				"Too many search results",
				fmt.Sprintf("Please refine your search to be more specific. Found %v persistent storages.", len(persistentStorages)),
			)
			return
		}

		persistentStorage := persistentStorages[0]

		// map response body to attributes
		if persistentStorage.Cloud != nil {
			data.CloudID = types.StringValue(persistentStorage.Cloud.ID)
		}
		data.ID = types.StringValue(persistentStorage.ID)
		data.Name = types.StringValue(persistentStorage.Name)
		data.Size = types.Int64Value(int64(persistentStorage.Capacity))
		if persistentStorage.Tenant != nil {
			data.TenantID = types.StringValue(persistentStorage.Tenant.ID)
		}
		data.UUID = types.StringValue(persistentStorage.UUID)
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
