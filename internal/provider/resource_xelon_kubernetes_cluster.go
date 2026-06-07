package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource              = (*kubernetesClusterResource)(nil)
	_ resource.ResourceWithConfigure = (*kubernetesClusterResource)(nil)
)

const (
	defaultKubernetesClusterTimeout = 30 * time.Minute
)

// kubernetesClusterResource is the Kubernetes cluster resource implementation.
type kubernetesClusterResource struct {
	client *xelon.Client
}

// kubernetesClusterResourceModel maps the Kubernetes cluster resource schema data.
type kubernetesClusterResourceModel struct {
	CloudID           types.String   `tfsdk:"cloud_id"`
	ControlPlane      types.Object   `tfsdk:"control_plane"` // kubernetesClusterNodeSpecResourceModel
	ID                types.String   `tfsdk:"id"`
	KubeConfigRaw     types.String   `tfsdk:"kube_config_raw"`
	KubernetesVersion types.String   `tfsdk:"kubernetes_version"`
	LoadBalancer      types.Object   `tfsdk:"load_balancer"` // kubernetesClusterNodeSpecResourceModel
	Name              types.String   `tfsdk:"name"`
	TalosVersion      types.String   `tfsdk:"talos_version"`
	TenantID          types.String   `tfsdk:"tenant_id"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

type kubernetesClusterNodeSpecResourceModel struct {
	CPUCoreCount            types.Int64 `tfsdk:"cpu_core_count"`
	DiskSize                types.Int64 `tfsdk:"disk_size"`
	HighAvailabilityEnabled types.Bool  `tfsdk:"high_availability_enabled"`
	Memory                  types.Int64 `tfsdk:"memory"`
}

func NewKubernetesClusterResource() resource.Resource { return &kubernetesClusterResource{} }

func (r *kubernetesClusterResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_kubernetes_cluster"
}

func (r *kubernetesClusterResource) Schema(ctx context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The kubernetes cluster resource allows you to manage Xelon Kubernetes (XKS) cluster.
XKS is a Kubernetes service with a fully managed control plane and high availability.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud in which the Kubernetes cluster will be provisioned.",
				Required:            true,
			},
			"control_plane": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration related to the cluster control plane.",
				Optional:            true,
				Computed:            true,
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					nodeSpecAttributeTypes(),
					nodeSpecDefaultValues(),
				)),
				Attributes: map[string]schema.Attribute{
					"cpu_core_count": schema.Int64Attribute{
						MarkdownDescription: "The number of CPU cores to allocate to control plane nodes. Defaults to `2`.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(2),
					},
					"disk_size": schema.Int64Attribute{
						MarkdownDescription: "The size of the primary disk in GB. Defaults to `50`.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(50),
					},
					"high_availability_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable high availability (HA) mode. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"memory": schema.Int64Attribute{
						MarkdownDescription: "The amount of RAM in GB to allocate to control plane nodes. Defaults to `4`.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(4),
					},
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Kubernetes cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kube_config_raw": schema.StringAttribute{
				MarkdownDescription: "Raw Kubernetes config to be used by kubectl and other compatible tools.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kubernetes_version": schema.StringAttribute{
				MarkdownDescription: "Desired Kubernetes version for the cluster.",
				Required:            true,
			},
			"load_balancer": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration related to the load balancer.",
				Optional:            true,
				Computed:            true,
				Default: objectdefault.StaticValue(types.ObjectValueMust(
					nodeSpecAttributeTypes(),
					nodeSpecDefaultValues(),
				)),
				Attributes: map[string]schema.Attribute{
					"cpu_core_count": schema.Int64Attribute{
						MarkdownDescription: "The number of CPU cores to allocate to the load balancer. Defaults to `2`.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(2),
					},
					"disk_size": schema.Int64Attribute{
						MarkdownDescription: "The size of the primary disk in GB. Defaults to `50`.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(50),
					},
					"high_availability_enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether to enable high availability (HA) mode. Defaults to `true`.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(true),
					},
					"memory": schema.Int64Attribute{
						MarkdownDescription: "The amount of RAM in GB to allocate to the load balancer. Defaults to `4`.",
						Optional:            true,
						Computed:            true,
						Default:             int64default.StaticInt64(4),
					},
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Kubernetes cluster.",
				Required:            true,
			},
			"talos_version": schema.StringAttribute{
				MarkdownDescription: "Desired Talos version for the Kubernetes cluster.",
				Required:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID of the Kubernetes cluster.",
				Required:            true,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: "Defaults to 30m.",
			}),
		},
	}
}

func (r *kubernetesClusterResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *kubernetesClusterResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data kubernetesClusterResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// configure timeout
	createTimeout, diags := data.Timeouts.Create(ctx, defaultKubernetesClusterTimeout)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	createRequest := &xelon.KubernetesClusterCreateRequest{
		CloudID:           data.CloudID.ValueString(),
		KubernetesVersion: data.KubernetesVersion.ValueString(),
		Name:              data.Name.ValueString(),
		TalosVersion:      data.TalosVersion.ValueString(),
		TenantID:          data.TenantID.ValueString(),

		// hardcoded by backend for now
		PodCIDRBlock:     "100.111.0.0/16",
		ServiceCIDRBlock: "100.112.0.0/12",
	}

	// handle optional defaults for control plane
	var controlPlaneModel kubernetesClusterNodeSpecResourceModel
	response.Diagnostics.Append(data.ControlPlane.As(ctx, &controlPlaneModel, basetypes.ObjectAsOptions{})...)
	if response.Diagnostics.HasError() {
		return
	}
	createRequest.ControlPlaneCPUCores = int(controlPlaneModel.CPUCoreCount.ValueInt64())
	createRequest.ControlPlaneDiskSize = int(controlPlaneModel.DiskSize.ValueInt64())
	createRequest.ControlPlaneRAM = int(controlPlaneModel.Memory.ValueInt64())
	createRequest.ControlPlaneCPUCores = int(controlPlaneModel.CPUCoreCount.ValueInt64())
	if controlPlaneModel.HighAvailabilityEnabled.ValueBool() {
		createRequest.ControlPlaneType = "production"
	} else {
		createRequest.ControlPlaneType = "test"
	}

	// load balancer configuration
	var loadBalancerModel kubernetesClusterNodeSpecResourceModel
	response.Diagnostics.Append(data.LoadBalancer.As(ctx, &loadBalancerModel, basetypes.ObjectAsOptions{})...)
	if response.Diagnostics.HasError() {
		return
	}
	createRequest.LoadBalancerCPUCores = int(loadBalancerModel.CPUCoreCount.ValueInt64())
	createRequest.LoadBalancerDiskSize = int(loadBalancerModel.DiskSize.ValueInt64())
	createRequest.LoadBalancerRAM = int(loadBalancerModel.Memory.ValueInt64())
	createRequest.LoadBalancerCPUCores = int(loadBalancerModel.CPUCoreCount.ValueInt64())
	if loadBalancerModel.HighAvailabilityEnabled.ValueBool() {
		createRequest.LoadBalancerType = "production"
	} else {
		createRequest.LoadBalancerType = "test"
	}

	tflog.Debug(ctx, "Creating Kubernetes cluster", map[string]any{"payload": createRequest})
	createdKubernetesCluster, _, err := r.client.Kubernetes.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create Kubernetes cluster", err.Error())
		return
	}
	tflog.Debug(ctx, "Created Kubernetes cluster", map[string]any{"data": createdKubernetesCluster})

	kubernetesClusterID := createdKubernetesCluster.ID

	tflog.Info(ctx, "Waiting for Kubernetes cluster to be ready", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	err = helper.WaitKubernetesClusterStatusReady(ctx, r.client, kubernetesClusterID, createTimeout)
	if err != nil {
		// set id to state that the resource will be marked as tainted
		response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), kubernetesClusterID)...)
		response.Diagnostics.AddError("Unable to wait for Kubernetes cluster to be ready", err.Error())
		return
	}

	tflog.Info(ctx, "Waiting for Kubernetes cluster to be healthy", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	err = helper.WaitKubernetesClusterControlPlaneStatusHealthy(ctx, r.client, kubernetesClusterID, createTimeout)
	if err != nil {
		// set id to state that the resource will be marked as tainted
		response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), kubernetesClusterID)...)
		response.Diagnostics.AddError("Unable to wait for Kubernetes cluster to be healthy", err.Error())
		return
	}

	tflog.Debug(ctx, "Getting Kubernetes control plane data", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	controlPlane, _, err := r.client.Kubernetes.ListControlPlane(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get Kubernetes cluster control plane", err.Error())
		return
	}
	tflog.Debug(ctx, "Got Kubernetes control plane data", map[string]any{"data": controlPlane})

	tflog.Debug(ctx, "Getting Kubernetes load balancer data", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	loadBalancer, _, err := r.client.Kubernetes.ListLoadBalancer(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get Kubernetes cluster load balancer", err.Error())
		return
	}
	tflog.Debug(ctx, "Got Kubernetes load balancer data", map[string]any{"data": controlPlane})

	tflog.Debug(ctx, "Getting kubeconfig", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	kubeConfig, _, err := r.client.Kubernetes.GetKubeConfig(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get kubeconfig", err.Error())
		return
	}
	tflog.Debug(ctx, "Got kubeconfig", map[string]any{"data": kubeConfig})

	// map response body to attributes
	data.ID = types.StringValue(kubernetesClusterID)
	data.ControlPlane, diags = flattenControlPlane(ctx, controlPlane)
	response.Diagnostics.Append(diags...)
	data.LoadBalancer, diags = flattenLoadBalancer(ctx, loadBalancer)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	data.KubeConfigRaw = types.StringValue(string(kubeConfig))

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *kubernetesClusterResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data kubernetesClusterResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	kubernetesClusterID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting Kubernetes cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	kubernetesCluster, _, err := r.client.Kubernetes.Get(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get Kubernetes cluster", err.Error())
		return
	}
	tflog.Debug(ctx, "Got Kubernetes cluster", map[string]any{"data": kubernetesCluster})

	tflog.Debug(ctx, "Getting Kubernetes control plane data", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	controlPlane, _, err := r.client.Kubernetes.ListControlPlane(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get Kubernetes cluster control plane", err.Error())
		return
	}
	tflog.Debug(ctx, "Got Kubernetes control plane data", map[string]any{"data": controlPlane})

	tflog.Debug(ctx, "Getting Kubernetes load balancer data", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	loadBalancer, _, err := r.client.Kubernetes.ListLoadBalancer(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get Kubernetes cluster load balancer", err.Error())
		return
	}
	tflog.Debug(ctx, "Got Kubernetes load balancer data", map[string]any{"data": controlPlane})

	tflog.Debug(ctx, "Getting kubeconfig", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	kubeConfig, _, err := r.client.Kubernetes.GetKubeConfig(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get kubeconfig", err.Error())
		return
	}
	tflog.Debug(ctx, "Got kubeconfig", map[string]any{"data": kubeConfig})

	// map response body to attributes
	data.ID = types.StringValue(kubernetesClusterID)
	data.ControlPlane, diags = flattenControlPlane(ctx, controlPlane)
	response.Diagnostics.Append(diags...)
	data.LoadBalancer, diags = flattenLoadBalancer(ctx, loadBalancer)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	data.KubeConfigRaw = types.StringValue(string(kubeConfig))

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *kubernetesClusterResource) Update(ctx context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	tflog.Info(ctx, "Update is not implemented yet")
}

func (r *kubernetesClusterResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data kubernetesClusterResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	kubernetesClusterID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting Kubernetes cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	_, err := r.client.Kubernetes.Delete(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete Kubernetes cluster", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted Kubernetes cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
}

func nodeSpecAttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cpu_core_count":            types.Int64Type,
		"disk_size":                 types.Int64Type,
		"high_availability_enabled": types.BoolType,
		"memory":                    types.Int64Type,
	}
}

func nodeSpecDefaultValues() map[string]attr.Value {
	return map[string]attr.Value{
		"cpu_core_count":            types.Int64Value(2),
		"disk_size":                 types.Int64Value(50),
		"high_availability_enabled": types.BoolValue(true),
		"memory":                    types.Int64Value(4),
	}
}

func flattenControlPlane(ctx context.Context, controlPlane *xelon.KubernetesClusterControlPlane) (types.Object, diag.Diagnostics) {
	highAvailabilityEnabled := len(controlPlane.Nodes) > 1
	return types.ObjectValueFrom(ctx, nodeSpecAttributeTypes(), kubernetesClusterNodeSpecResourceModel{
		CPUCoreCount:            types.Int64Value(int64(controlPlane.CPUCores)),
		DiskSize:                types.Int64Value(int64(controlPlane.DiskSize)),
		HighAvailabilityEnabled: types.BoolValue(highAvailabilityEnabled),
		Memory:                  types.Int64Value(int64(controlPlane.RAM)),
	})
}

func flattenLoadBalancer(ctx context.Context, loadBalancer *xelon.KubernetesClusterLoadBalancer) (types.Object, diag.Diagnostics) {
	highAvailabilityEnabled := len(loadBalancer.Instances) > 1
	return types.ObjectValueFrom(ctx, nodeSpecAttributeTypes(), kubernetesClusterNodeSpecResourceModel{
		CPUCoreCount:            types.Int64Value(int64(loadBalancer.CPUCores)),
		DiskSize:                types.Int64Value(int64(loadBalancer.DiskSize)),
		HighAvailabilityEnabled: types.BoolValue(highAvailabilityEnabled),
		Memory:                  types.Int64Value(int64(loadBalancer.RAM)),
	})
}
