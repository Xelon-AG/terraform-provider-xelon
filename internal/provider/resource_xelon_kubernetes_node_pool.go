package provider

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider/helper"
	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

var (
	_ resource.Resource              = (*kubernetesNodePoolResource)(nil)
	_ resource.ResourceWithConfigure = (*kubernetesNodePoolResource)(nil)
)

const (
	defaultKubernetesNodePoolTimeout = 30 * time.Minute
)

// kubernetesNodePoolResource is the Kubernetes node pool resource implementation.
type kubernetesNodePoolResource struct {
	client *xelon.Client
}

// kubernetesNodePoolResourceModel maps the Kubernetes ode pool resource schema data.
type kubernetesNodePoolResourceModel struct {
	CPUCoreCount        types.Int64    `tfsdk:"cpu_core_count"`
	DiskSize            types.Int64    `tfsdk:"disk_size"`
	ID                  types.String   `tfsdk:"id"`
	KubernetesClusterID types.String   `tfsdk:"kubernetes_cluster_id"`
	Memory              types.Int64    `tfsdk:"memory"`
	Name                types.String   `tfsdk:"name"`
	NodeCount           types.Int64    `tfsdk:"node_count"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

func NewKubernetesNodePoolResource() resource.Resource {
	return &kubernetesNodePoolResource{}
}

func (r *kubernetesNodePoolResource) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "xelon_kubernetes_node_pool"
}

func (r *kubernetesNodePoolResource) Schema(ctx context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: `
The kubernetes node pool resource allows you to manage node pools in Xelon Kubernetes (XKS) cluster.
XKS is a Kubernetes service with a fully managed control plane and high availability.
`,
		Version: 0,
		Attributes: map[string]schema.Attribute{
			"cpu_core_count": schema.Int64Attribute{
				MarkdownDescription: "The number of CPU cores to allocate to pool nodes. Defaults to `2`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2),
			},
			"disk_size": schema.Int64Attribute{
				MarkdownDescription: "The size of the primary disk in GB to allocate to pool nodes. Defaults to `50`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(50),
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the node pool.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kubernetes_cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Kubernetes cluster to which the node pool is associated.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"memory": schema.Int64Attribute{
				MarkdownDescription: "The amount of RAM in GB to allocate to pool nodes. Defaults to `4`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(4),
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the Kubernetes node pool.",
				Required:            true,
			},
			"node_count": schema.Int64Attribute{
				MarkdownDescription: "The number of nodes in pool. Defaults to `3`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3),
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: "Defaults to 30m.",
				Update:            true,
				UpdateDescription: "Defaults to 30m.",
			}),
		},
	}
}

func (r *kubernetesNodePoolResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *kubernetesNodePoolResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data kubernetesNodePoolResourceModel

	// read plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// configure timeout
	createTimeout, diags := data.Timeouts.Create(ctx, defaultKubernetesNodePoolTimeout)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	kubernetesClusterID := data.KubernetesClusterID.ValueString()
	createRequest := &xelon.KubernetesClusterNodePoolCreateRequest{
		CPUCores:             int(data.CPUCoreCount.ValueInt64()),
		DiskSize:             int(data.DiskSize.ValueInt64()),
		ExtraStorageDiskSize: 0,
		ExtraStorageEnabled:  false,
		Name:                 data.Name.ValueString(),
		NodeCount:            int(data.NodeCount.ValueInt64()),
		RAM:                  int(data.Memory.ValueInt64()),
	}
	tflog.Debug(ctx, "Creating Kubernetes node pool", map[string]any{"payload": createRequest})
	createdNodePool, _, err := r.client.Kubernetes.CreateNodePool(ctx, kubernetesClusterID, createRequest)
	if err != nil {
		response.Diagnostics.AddError("Unable to create Kubernetes node pool", err.Error())
		return
	}
	tflog.Debug(ctx, "Created Kubernetes nodes pool", map[string]any{"data": createdNodePool})

	nodePoolID := createdNodePool.ID

	tflog.Info(ctx, "Waiting for Kubernetes node pool to be ready", map[string]any{
		"kubernetes_cluster_id": kubernetesClusterID,
		"node_pool_id":          nodePoolID,
	})
	err = helper.WaitKubernetesNodePoolStatusReady(ctx, r.client, kubernetesClusterID, nodePoolID, createTimeout)
	if err != nil {
		// set id to state that the resource will be marked as tainted
		response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), nodePoolID)...)
		response.Diagnostics.AddError("Unable to wait for Kubernetes node pool to be ready", err.Error())
		return
	}
	tflog.Info(ctx, "Kubernetes node pool is ready", map[string]any{
		"kubernetes_cluster_id": kubernetesClusterID,
		"node_pool_id":          nodePoolID,
	})

	tflog.Debug(ctx, "Getting Kubernetes node pool with enriched properties", map[string]any{
		"kubernetes_cluster_id": kubernetesClusterID,
		"node_pool_id":          nodePoolID,
	})
	nodePool, _, err := r.client.Kubernetes.GetNodePool(ctx, kubernetesClusterID, nodePoolID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get Kubernetes node pool", err.Error())
		return
	}
	tflog.Debug(ctx, "Got Kubernetes node pool with enriched properties", map[string]any{"data": nodePool})

	// map response body to attributes
	data.fromAPI(nodePool, kubernetesClusterID)
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *kubernetesNodePoolResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data kubernetesNodePoolResourceModel

	// read plan data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	kubernetesClusterID := data.KubernetesClusterID.ValueString()
	nodePoolID := data.ID.ValueString()
	tflog.Debug(ctx, "Getting Kubernetes node pool", map[string]any{
		"kubernetes_cluster_id": kubernetesClusterID,
		"node_pool_id":          nodePoolID,
	})
	nodePool, resp, err := r.client.Kubernetes.GetNodePool(ctx, kubernetesClusterID, nodePoolID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the node pool is somehow already destroyed, mark as successfully gone
			response.State.RemoveResource(ctx)
			return
		}
		response.Diagnostics.AddError("Unable to get node pool", err.Error())
		return
	}
	tflog.Debug(ctx, "Got Kubernetes node pool", map[string]any{"data": nodePool})

	// map response body to attributes
	data.fromAPI(nodePool, kubernetesClusterID)
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
}

func (r *kubernetesNodePoolResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state kubernetesNodePoolResourceModel

	// read plan and state data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	// configure timeout
	updateTimeout, diags := plan.Timeouts.Update(ctx, defaultKubernetesNodePoolTimeout)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	kubernetesClusterID := state.KubernetesClusterID.ValueString()
	nodePoolID := state.ID.ValueString()

	if !plan.NodeCount.Equal(state.NodeCount) {
		planNodeCount := int(plan.NodeCount.ValueInt64())
		stateNodeCount := int(state.NodeCount.ValueInt64())

		if planNodeCount > stateNodeCount {
			// add delta nodes
			delta := planNodeCount - stateNodeCount
			for range delta {
				_, err := r.client.Kubernetes.CreateNode(ctx, kubernetesClusterID, nodePoolID)
				if err != nil {
					response.Diagnostics.AddError("Unable to add node to pool", err.Error())
					return
				}
			}
		} else if planNodeCount < stateNodeCount {
			nodePool, _, err := r.client.Kubernetes.GetNodePool(ctx, kubernetesClusterID, nodePoolID)
			if err != nil {
				response.Diagnostics.AddError("Unable to get node pool", err.Error())
				return
			}

			var removableNodeIDs []string
			for _, node := range nodePool.Nodes {
				if node.Status == "Deployed" {
					removableNodeIDs = append(removableNodeIDs, node.ID)
				}
			}

			delta := stateNodeCount - planNodeCount
			if len(removableNodeIDs) < delta {
				response.Diagnostics.AddError(
					"Unable to remove nodes from pool",
					fmt.Sprintf("Need to remove %d node(s) but only %d node(s) available", delta, len(removableNodeIDs)),
				)
				return
			}

			slices.Reverse(removableNodeIDs)
			for _, nodeIDToRemove := range removableNodeIDs[:delta] {
				_, err := r.client.Kubernetes.DeleteNode(ctx, kubernetesClusterID, nodeIDToRemove)
				if err != nil {
					response.Diagnostics.AddError("Unable to remove node from pool", err.Error())
					return
				}
			}
		}

		tflog.Info(ctx, "Waiting for Kubernetes node pool to be ready", map[string]any{
			"kubernetes_cluster_id": kubernetesClusterID,
			"node_pool_id":          nodePoolID,
		})
		err := helper.WaitKubernetesNodePoolStatusReady(ctx, r.client, kubernetesClusterID, nodePoolID, updateTimeout)
		if err != nil {
			// set id to state that the resource will be marked as tainted
			response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), nodePoolID)...)
			response.Diagnostics.AddError("Unable to wait for Kubernetes node pool to be ready", err.Error())
			return
		}
		tflog.Info(ctx, "Kubernetes node pool is ready", map[string]any{
			"kubernetes_cluster_id": kubernetesClusterID,
			"node_pool_id":          nodePoolID,
		})
	}

	// update worker pool only if attributes changed
	if !plan.CPUCoreCount.Equal(state.CPUCoreCount) ||
		!plan.DiskSize.Equal(state.DiskSize) ||
		!plan.Name.Equal(state.Name) ||
		!plan.Memory.Equal(state.Memory) {
		updateRequest := &xelon.KubernetesClusterNodePoolUpdateRequest{
			CPUCores:             int(plan.CPUCoreCount.ValueInt64()),
			DiskSize:             int(plan.DiskSize.ValueInt64()),
			ExtraStorageEnabled:  0,
			ExtraStorageDiskSize: 0,
			Name:                 plan.Name.ValueString(),
			RAM:                  int(plan.Memory.ValueInt64()),
		}
		tflog.Debug(ctx, "Updating Kubernetes node pool", map[string]any{
			"kubernetes_cluster_id": kubernetesClusterID,
			"node_pool_id":          nodePoolID,
			"payload":               updateRequest,
		})
		_, err := r.client.Kubernetes.UpdateNodePool(ctx, kubernetesClusterID, nodePoolID, updateRequest)
		if err != nil {
			response.Diagnostics.AddError("Unable to update Kubernetes node pool", err.Error())
			return
		}
		tflog.Debug(ctx, "Updated Kubernetes node pool", map[string]any{
			"kubernetes_cluster_id": kubernetesClusterID,
			"node_pool_id":          nodePoolID,
		})

		tflog.Info(ctx, "Waiting for Kubernetes node pool to be ready", map[string]any{
			"kubernetes_cluster_id": kubernetesClusterID,
			"node_pool_id":          nodePoolID,
		})
		err = helper.WaitKubernetesNodePoolStatusReady(ctx, r.client, kubernetesClusterID, nodePoolID, updateTimeout)
		if err != nil {
			// set id to state that the resource will be marked as tainted
			response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), nodePoolID)...)
			response.Diagnostics.AddError("Unable to wait for Kubernetes node pool to be ready", err.Error())
			return
		}
		tflog.Info(ctx, "Kubernetes node pool is ready", map[string]any{
			"kubernetes_cluster_id": kubernetesClusterID,
			"node_pool_id":          nodePoolID,
		})
	}

	tflog.Debug(ctx, "Getting Kubernetes node pool with enriched properties", map[string]any{
		"kubernetes_cluster_id": kubernetesClusterID,
		"node_pool_id":          nodePoolID,
	})
	nodePool, _, err := r.client.Kubernetes.GetNodePool(ctx, kubernetesClusterID, nodePoolID)
	if err != nil {
		response.Diagnostics.AddError("Unable to get Kubernetes node pool", err.Error())
		return
	}
	tflog.Debug(ctx, "Got Kubernetes node pool", map[string]any{"data": nodePool})

	// map response body to attributes
	plan.fromAPI(nodePool, kubernetesClusterID)
	diags = response.State.Set(ctx, &plan)
	response.Diagnostics.Append(diags...)
}

func (r *kubernetesNodePoolResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data kubernetesNodePoolResourceModel

	// read state data into the model
	diags := request.State.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	kubernetesClusterID := data.KubernetesClusterID.ValueString()
	nodePoolID := data.ID.ValueString()
	tflog.Debug(ctx, "Deleting Kubernetes node pool", map[string]any{
		"kubernetes_cluster_id": kubernetesClusterID,
		"node_pool_id":          nodePoolID,
	})
	_, err := r.client.Kubernetes.DeleteNodePool(ctx, kubernetesClusterID, nodePoolID)
	if err != nil {
		response.Diagnostics.AddError("Unable to delete Kubernetes node pool", err.Error())
		return
	}
	tflog.Debug(ctx, "Deleted Kubernetes node pool", map[string]any{
		"kubernetes_cluster_id": kubernetesClusterID,
		"node_pool_id":          nodePoolID,
	})
}

func (m *kubernetesNodePoolResourceModel) fromAPI(nodePool *xelon.KubernetesClusterNodePool, kubernetesClusterID string) {
	m.CPUCoreCount = types.Int64Value(int64(nodePool.CPUCores))
	m.DiskSize = types.Int64Value(int64(nodePool.DiskSize))
	m.ID = types.StringValue(nodePool.ID)
	m.KubernetesClusterID = types.StringValue(kubernetesClusterID)
	m.Memory = types.Int64Value(int64(nodePool.RAM))
	m.Name = types.StringValue(nodePool.Name)
	m.NodeCount = types.Int64Value(int64(len(nodePool.Nodes)))
}
