package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	kubernetesNodePoolStatusProvisioning = "provisioning"
	kubernetesNodePoolStatusReady        = "ready"
)

func WaitKubernetesNodePoolStatusReady(ctx context.Context, client *xelon.Client, kubernetesClusterID, nodePoolID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{kubernetesNodePoolStatusProvisioning},
		Target:     []string{kubernetesNodePoolStatusReady},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    stateKubernetesNodePoolStatus(ctx, client, kubernetesClusterID, nodePoolID),
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for kubernetes node pool (%s) to become ready: %w", nodePoolID, err)
	}
	return nil
}

func stateKubernetesNodePoolStatus(ctx context.Context, client *xelon.Client, kubernetesClusterID, nodePoolID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		nodePool, _, err := client.Kubernetes.GetNodePool(ctx, kubernetesClusterID, nodePoolID)
		if err != nil {
			return nil, "", err
		}
		if nodePool == nil {
			return nil, "", fmt.Errorf("failed to get kubernetes node pool with id: %s", nodePoolID)
		}

		// pool without nodes is considered as ready
		if len(nodePool.Nodes) == 0 {
			return nodePool, kubernetesNodePoolStatusReady, nil
		}

		for _, node := range nodePool.Nodes {
			tflog.Debug(ctx, "Checking node pool status", map[string]any{"node": node})
			if node.Status != "Deployed" {
				return nodePool, kubernetesNodePoolStatusProvisioning, nil
			}
		}

		return nodePool, kubernetesNodePoolStatusReady, nil
	}
}
