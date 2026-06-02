package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ datasource.DataSource              = (*xksClusterVersionsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*xksClusterVersionsDataSource)(nil)
)

// xksClusterVersionsDataSource is the XKS cluster versions data source implementation.
type xksClusterVersionsDataSource struct {
	client *xelon.Client
}

// xksClusterVersionsDataSourceModel maps the  XKS cluster versions datasource schema data.
type xksClusterVersionsDataSourceModel struct {
	CloudID  types.String                              `tfsdk:"cloud_id"`
	Latest   types.Object                              `tfsdk:"latest"` // xksClusterLatestDataSourceModel
	Versions []kubernetesByTalosVersionDataSourceModel `tfsdk:"versions"`
}

type xksClusterLatestDataSourceModel struct {
	TalosVersion      types.String `tfsdk:"talos_version"`
	KubernetesVersion types.String `tfsdk:"kubernetes_version"`
}

type kubernetesByTalosVersionDataSourceModel struct {
	TalosVersion       string   `tfsdk:"talos_version"`
	KubernetesVersions []string `tfsdk:"kubernetes_versions"`
}

func NewXKSClusterVersionsDataSource() datasource.DataSource {
	return &xksClusterVersionsDataSource{}
}

func (d *xksClusterVersionsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_xks_cluster_versions"
}

func (d *xksClusterVersionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The XKS cluster versions data source provides information about available Talos and Kubernetes versions.
`,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud.",
				Required:            true,
			},
			"latest": schema.SingleNestedAttribute{
				MarkdownDescription: "The latest Talos version and compatible Kubernetes version.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"kubernetes_version": schema.StringAttribute{
						MarkdownDescription: "The latest Kubernetes version available for the specific Talos version.",
						Computed:            true,
					},
					"talos_version": schema.StringAttribute{
						MarkdownDescription: "The latest Talos version used to run Kubernetes cluster.",
						Computed:            true,
					},
				},
			},
			"versions": schema.ListNestedAttribute{
				MarkdownDescription: "The mapping of compatible Talos Linux versions to Kubernetes versions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kubernetes_versions": schema.ListAttribute{
							MarkdownDescription: "The list of Kubernetes versions available for the specific Talos version.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"talos_version": schema.StringAttribute{
							MarkdownDescription: "The Talos version used to run Kubernetes cluster.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *xksClusterVersionsDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *xksClusterVersionsDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data xksClusterVersionsDataSourceModel

	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	cloudID := data.CloudID.ValueString()

	tflog.Debug(ctx, "Getting cluster version mappings", map[string]any{"cloud_id": cloudID})
	versionMapping, _, err := d.client.Kubernetes.ListVersionMapping(ctx, cloudID)
	if err != nil {
		response.Diagnostics.AddError("Unable to list cluster versions", err.Error())
		return
	}
	tflog.Debug(ctx, "Got cluster version mappings", map[string]any{"data": versionMapping})

	// map response body to attributes
	var versions []kubernetesByTalosVersionDataSourceModel
	talosVersions := make([]string, 0, len(versionMapping))
	for talosVersion, k8sVersions := range versionMapping {
		// copy to not mutate API response, and then sort newest first
		k8sVersionsSorted := append([]string(nil), k8sVersions...)
		helper.SortVersions(k8sVersionsSorted, func(s string) string { return s })

		talosVersions = append(talosVersions, talosVersion)
		versions = append(versions, kubernetesByTalosVersionDataSourceModel{
			TalosVersion:       talosVersion,
			KubernetesVersions: k8sVersionsSorted,
		})
	}
	data.Versions = versions

	// latest pair of talos and corresponding kubernetes version
	var latest = xksClusterLatestDataSourceModel{}
	helper.SortVersions(talosVersions, func(s string) string { return s })
	if len(talosVersions) > 0 {
		latest.TalosVersion = types.StringValue(talosVersions[0])
		k8sVersions, ok := versionMapping[talosVersions[0]]
		if ok {
			k8sVersionsSorted := append([]string(nil), k8sVersions...)
			helper.SortVersions(k8sVersionsSorted, func(s string) string { return s })
			latest.KubernetesVersion = types.StringValue(k8sVersionsSorted[0])
		}
	}
	latestAttributeTypes := map[string]attr.Type{
		"talos_version":      types.StringType,
		"kubernetes_version": types.StringType,
	}
	data.Latest, diags = types.ObjectValueFrom(ctx, latestAttributeTypes, latest)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
