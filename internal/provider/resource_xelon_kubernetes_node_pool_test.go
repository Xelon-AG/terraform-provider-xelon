package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestKubernetesNodePoolResourceModel_fromAPI_withoutNodes(t *testing.T) {
	nodePool := &xelon.KubernetesClusterNodePool{
		CPUCores: 2,
		DiskSize: 50,
		ID:       "node-pool-id",
		Name:     "node-pool-name",
		RAM:      4,
	}
	expectedModel := kubernetesNodePoolResourceModel{
		CPUCoreCount:        types.Int64Value(2),
		DiskSize:            types.Int64Value(50),
		ID:                  types.StringValue("node-pool-id"),
		KubernetesClusterID: types.StringValue("kubernetes-cluster-id"),
		Name:                types.StringValue("node-pool-name"),
		NodeCount:           types.Int64Value(0),
		Memory:              types.Int64Value(4),
	}

	var actualModel kubernetesNodePoolResourceModel
	actualModel.fromAPI(nodePool, "kubernetes-cluster-id")

	assert.Equal(t, expectedModel, actualModel)
}

func TestKubernetesNodePoolResourceModel_fromAPI_withNodes(t *testing.T) {
	nodePool := &xelon.KubernetesClusterNodePool{
		CPUCores: 2,
		DiskSize: 50,
		ID:       "node-pool-id",
		Name:     "node-pool-name",
		Nodes: []xelon.KubernetesClusterNode{
			{ID: "1111", Name: "node-w-1", Status: "Deployed"},
			{ID: "2222", Name: "node-w-2", Status: "Created"},
		},
		RAM: 4,
	}
	expectedModel := kubernetesNodePoolResourceModel{
		CPUCoreCount:        types.Int64Value(2),
		DiskSize:            types.Int64Value(50),
		ID:                  types.StringValue("node-pool-id"),
		KubernetesClusterID: types.StringValue("kubernetes-cluster-id"),
		Name:                types.StringValue("node-pool-name"),
		NodeCount:           types.Int64Value(2),
		Memory:              types.Int64Value(4),
	}

	var actualModel kubernetesNodePoolResourceModel
	actualModel.fromAPI(nodePool, "kubernetes-cluster-id")

	assert.Equal(t, expectedModel, actualModel)
}
