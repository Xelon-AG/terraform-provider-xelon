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
	_ resource.Resource              = (*xksClusterResource)(nil)
	_ resource.ResourceWithConfigure = (*xksClusterResource)(nil)
)

const (
	defaultXKSClusterTimeout = 30 * time.Minute
)

// xksClusterResource is the XKS cluster resource implementation.
type xksClusterResource struct {
	client *xelon.Client
}

// xksClusterResourceModel maps the XKS cluster resource schema data.
type xksClusterResourceModel struct {
	CloudID           types.String   `tfsdk:"cloud_id"`
	ControlPlane      types.Object   `tfsdk:"control_plane"` // xksClusterControlPlaneResourceModel
	ID                types.String   `tfsdk:"id"`
	KubernetesVersion types.String   `tfsdk:"kubernetes_version"`
	LoadBalancer      types.Object   `tfsdk:"load_balancer"` // xksClusterLoadBalancerResourceModel
	Name              types.String   `tfsdk:"name"`
	TalosVersion      types.String   `tfsdk:"talos_version"`
	TenantID          types.String   `tfsdk:"tenant_id"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

type xksClusterControlPlaneResourceModel struct {
	CPUCoreCount            types.Int64 `tfsdk:"cpu_core_count"`
	DiskSize                types.Int64 `tfsdk:"disk_size"`
	HighAvailabilityEnabled types.Bool  `tfsdk:"high_availability_enabled"`
	Memory                  types.Int64 `tfsdk:"memory"`
}

type xksClusterLoadBalancerResourceModel struct {
	CPUCoreCount            types.Int64 `tfsdk:"cpu_core_count"`
	DiskSize                types.Int64 `tfsdk:"disk_size"`
	HighAvailabilityEnabled types.Bool  `tfsdk:"high_availability_enabled"`
	Memory                  types.Int64 `tfsdk:"memory"`
}

func NewXKSClusterResource() resource.Resource { return &xksClusterResource{} }

func (r *xksClusterResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_xks_cluster"
}

func (r *xksClusterResource) Schema(ctx context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The XKS cluster resource allows you to manage Xelon Kubernetes cluster.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"cloud_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cloud in which the XKS cluster will be provisioned.",
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
				MarkdownDescription: "The ID of the XKS cluster.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kubernetes_version": schema.StringAttribute{
				MarkdownDescription: "Desired Kubernetes version for the XKS cluster.",
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
				MarkdownDescription: "The name of the XKS cluster.",
				Required:            true,
			},
			"talos_version": schema.StringAttribute{
				MarkdownDescription: "Desired Talos version for the XKS cluster.",
				Required:            true,
			},
			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The tenant ID of the XKS cluster.",
				Required:            true,
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: "Defaults to 30m.",
			}),
		},
	}
}

func (r *xksClusterResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *xksClusterResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data xksClusterResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// configure timeout
	createTimeout, diags := data.Timeouts.Create(ctx, defaultXKSClusterTimeout)
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
	var controlPlaneData xksClusterControlPlaneResourceModel
	response.Diagnostics.Append(data.ControlPlane.As(ctx, &controlPlaneData, basetypes.ObjectAsOptions{})...)
	if response.Diagnostics.HasError() {
		return
	}
	createRequest.ControlPlaneCPUCores = int(controlPlaneData.CPUCoreCount.ValueInt64())
	createRequest.ControlPlaneDiskSize = int(controlPlaneData.DiskSize.ValueInt64())
	createRequest.ControlPlaneRAM = int(controlPlaneData.Memory.ValueInt64())
	createRequest.ControlPlaneCPUCores = int(controlPlaneData.CPUCoreCount.ValueInt64())
	if controlPlaneData.HighAvailabilityEnabled.ValueBool() {
		createRequest.ControlPlaneType = "production"
	} else {
		createRequest.ControlPlaneType = "test"
	}

	// load balancer configuration
	var loadBalancer xksClusterLoadBalancerResourceModel
	response.Diagnostics.Append(data.LoadBalancer.As(ctx, &loadBalancer, basetypes.ObjectAsOptions{})...)
	if response.Diagnostics.HasError() {
		return
	}
	createRequest.LoadBalancerCPUCores = int(loadBalancer.CPUCoreCount.ValueInt64())
	createRequest.LoadBalancerDiskSize = int(loadBalancer.DiskSize.ValueInt64())
	createRequest.LoadBalancerRAM = int(loadBalancer.Memory.ValueInt64())
	createRequest.LoadBalancerCPUCores = int(loadBalancer.CPUCoreCount.ValueInt64())
	if loadBalancer.HighAvailabilityEnabled.ValueBool() {
		createRequest.LoadBalancerType = "production"
	} else {
		createRequest.LoadBalancerType = "test"
	}

	tflog.Debug(ctx, "Creating XKS cluster", map[string]any{"payload": createRequest})
	createdXKSCluster, _, err := r.client.Kubernetes.Create(ctx, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create XKS cluster", err.Error())
		return
	}
	tflog.Debug(ctx, "Created XKS cluster", map[string]any{"data": createdXKSCluster})

	kubernetesClusterID := createdXKSCluster.ID

	tflog.Info(ctx, "Waiting for XKS cluster to be ready", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	err = helper.WaitXKSClusterStatusReady(ctx, r.client, kubernetesClusterID, createTimeout)
	if err != nil {
		// set id to state that the resource will be marked as tainted
		response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), kubernetesClusterID)...)
		response.Diagnostics.AddError("Unable to wait for XKS cluster to be ready", err.Error())
		return
	}

	tflog.Info(ctx, "Waiting for XKS cluster to be healthy", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	err = helper.WaitXKSClusterControlPlaneStatusHealthy(ctx, r.client, kubernetesClusterID, createTimeout)
	if err != nil {
		// set id to state that the resource will be marked as tainted
		response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), kubernetesClusterID)...)
		response.Diagnostics.AddError("Unable to wait for XKS cluster to be healthy", err.Error())
		return
	}

	tflog.Debug(ctx, "Getting XKS control plane data", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	controlPlane, _, err := r.client.Kubernetes.ListControlPlane(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get XKS cluster control plane", err.Error())
		return
	}
	tflog.Debug(ctx, "Got XKS control plane data", map[string]any{"data": controlPlane})

	// map response body to attributes
	data.ControlPlane, diags = flattenControlPlane(ctx, controlPlane)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	data.ID = types.StringValue(kubernetesClusterID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *xksClusterResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data xksClusterResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	kubernetesClusterID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting XKS cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	kubernetesCluster, _, err := r.client.Kubernetes.Get(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get XKS cluster", err.Error())
		return
	}
	tflog.Debug(ctx, "Got XKS cluster", map[string]any{"data": kubernetesCluster})

	tflog.Debug(ctx, "Getting XKS control plane data", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	controlPlane, _, err := r.client.Kubernetes.ListControlPlane(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get XKS cluster control plane", err.Error())
		return
	}
	tflog.Debug(ctx, "Got XKS control plane data", map[string]any{"data": controlPlane})

	// map response body to attributes
	data.ControlPlane, diags = flattenControlPlane(ctx, controlPlane)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	data.ID = types.StringValue(kubernetesClusterID)

	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *xksClusterResource) Update(ctx context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	tflog.Info(ctx, "Update is not implemented yet")
}

func (r *xksClusterResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data xksClusterResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	kubernetesClusterID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting XKS cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
	_, err := r.client.Kubernetes.Delete(ctx, kubernetesClusterID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete XKS cluster", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted XKS cluster", map[string]any{"kubernetes_cluster_id": kubernetesClusterID})
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
	return types.ObjectValueFrom(ctx, nodeSpecAttributeTypes(), xksClusterControlPlaneResourceModel{
		CPUCoreCount:            types.Int64Value(int64(controlPlane.CPUCores)),
		DiskSize:                types.Int64Value(int64(controlPlane.DiskSize)),
		HighAvailabilityEnabled: types.BoolValue(highAvailabilityEnabled),
		Memory:                  types.Int64Value(int64(controlPlane.RAM)),
	})
}
