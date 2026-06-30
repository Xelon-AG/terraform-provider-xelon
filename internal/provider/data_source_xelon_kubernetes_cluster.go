package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ datasource.DataSource              = (*kubernetesClusterDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*kubernetesClusterDataSource)(nil)
)

// kubernetesClusterDataSource is the Kubernetes cluster data source implementation.
type kubernetesClusterDataSource struct {
	client *xelon.Client
}

// kubernetesClusterDataSourceModel maps the Kubernetes cluster datasource schema data.
type kubernetesClusterDataSourceModel struct {
	KubeconfigRaw       types.String `tfsdk:"kubeconfig_raw"`
	KubernetesClusterID types.String `tfsdk:"kubernetes_cluster_id"`
	TalosconfigRaw      types.String `tfsdk:"talosconfig_raw"`
}

func NewKubernetesClusterDataSource() datasource.DataSource {
	return &kubernetesClusterDataSource{}
}

func (d *kubernetesClusterDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = "xelon_kubernetes_cluster"
}

func (d *kubernetesClusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The Kubernetes cluster data source provides information about an existing Xelon Kubernetes (XKS) cluster.
XKS is a Kubernetes service with a fully managed control plane and high availability.
`,
		Attributes: map[string]schema.Attribute{
			"kubeconfig_raw": schema.StringAttribute{
				MarkdownDescription: "Raw Kubernetes config for this Kubernetes cluster to be used by " +
					"[kubectl](https://kubernetes.io/docs/reference/kubectl/overview/) and other compatible tools.",
				Computed:  true,
				Sensitive: true,
			},
			"kubernetes_cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Kubernetes cluster.",
				Required:            true,
			},
			"talosconfig_raw": schema.StringAttribute{
				MarkdownDescription: "Raw Talosconfig for this Kubernetes cluster to be used by " +
					"[talosctl](https://docs.siderolabs.com/talos/latest/reference/cli).",
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func (d *kubernetesClusterDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (d *kubernetesClusterDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data kubernetesClusterDataSourceModel
	diags := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	kubernetesClusterID := data.KubernetesClusterID.ValueString()

	tflog.Debug(ctx, "Getting kubeconfig for cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	kubeconfig, _, err := d.client.Kubernetes.GetKubeconfig(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get kubeconfig", err.Error())
		return
	}
	tflog.Debug(ctx, "Got kubeconfig for cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})

	tflog.Debug(ctx, "Getting talosconfig for cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	talosconfig, _, err := d.client.Kubernetes.GetTalosconfig(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get talosconfig", err.Error())
		return
	}
	tflog.Debug(ctx, "Got talosconfig for cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})

	// map response body to attributes
	data.KubeconfigRaw = types.StringValue(string(kubeconfig))
	data.TalosconfigRaw = types.StringValue(string(talosconfig))

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}
