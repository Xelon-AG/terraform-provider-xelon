package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	kubernetesClusterStatusProvisioning             = "provisioning"
	kubernetesClusterStatusReady                    = "ready"
	kubernetesClusterStatusControlPlaneHealthy      = "healthy"
	kubernetesClusterStatusControlPlaneProvisioning = "provisioning"
)

func WaitKubernetesClusterStatusReady(ctx context.Context, client *xelon.Client, kubernetesClusterID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{kubernetesClusterStatusProvisioning},
		Target:     []string{kubernetesClusterStatusReady},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    statusKubernetesClusterStatus(ctx, client, kubernetesClusterID),
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for kubernetes cluster (%s) to become ready: %w", kubernetesClusterID, err)
	}
	return nil
}

func statusKubernetesClusterStatus(ctx context.Context, client *xelon.Client, kubernetesClusterID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		kubernetesCluster, _, err := client.Kubernetes.Get(ctx, kubernetesClusterID)
		if err != nil {
			return nil, "", err
		}
		if kubernetesCluster == nil {
			return nil, "", fmt.Errorf("failed get to kubernetes cluster with id: %s", kubernetesClusterID)
		}

		switch kubernetesCluster.Status {
		case "Ready":
			return kubernetesCluster, kubernetesClusterStatusReady, nil
		case "Deleting", "Deleted", "Error":
			return nil, "", fmt.Errorf("kubernetes cluster %s entered terminal state %q while waiting for ready", kubernetesClusterID, kubernetesCluster.Status)
		default:
			return kubernetesCluster, kubernetesClusterStatusProvisioning, nil
		}
	}
}

func WaitKubernetesClusterControlPlaneStatusHealthy(ctx context.Context, client *xelon.Client, kubernetesClusterID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{kubernetesClusterStatusControlPlaneProvisioning},
		Target:     []string{kubernetesClusterStatusControlPlaneHealthy},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    statusKubernetesClusterControlPlaneStateStatus(ctx, client, kubernetesClusterID),
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for kubernetes cluster (%s) to become healthy: %w", kubernetesClusterID, err)
	}
	return nil
}

func statusKubernetesClusterControlPlaneStateStatus(ctx context.Context, client *xelon.Client, kubernetesClusterID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		kubernetesCluster, _, err := client.Kubernetes.Get(ctx, kubernetesClusterID)
		if err != nil {
			return nil, "", err
		}
		if kubernetesCluster == nil {
			return nil, "", fmt.Errorf("failed to get kubernetes cluster with id: %s", kubernetesClusterID)
		}
		if kubernetesCluster.Health == nil {
			return nil, "", fmt.Errorf("failed to kubernetes cluster %s health data", kubernetesClusterID)
		}

		if kubernetesCluster.Health.Status == "healthy" {
			return kubernetesCluster, kubernetesClusterStatusControlPlaneHealthy, nil
		}
		return kubernetesCluster, kubernetesClusterStatusControlPlaneProvisioning, nil
	}
}
