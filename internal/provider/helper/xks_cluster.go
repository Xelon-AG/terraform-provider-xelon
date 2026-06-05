package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

const (
	xksClusterStatusProvisioning             = "provisioning"
	xksClusterStatusReady                    = "ready"
	xksClusterStatusControlPlaneHealthy      = "healthy"
	xksClusterStatusControlPlaneProvisioning = "provisioning"
)

func WaitXKSClusterStatusReady(ctx context.Context, client *xelon.Client, kubernetesClusterID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{xksClusterStatusProvisioning},
		Target:     []string{xksClusterStatusReady},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    statusXKSClusterStatus(ctx, client, kubernetesClusterID),
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for kubernetes cluster (%s) to become ready: %w", kubernetesClusterID, err)
	}
	return nil
}

func statusXKSClusterStatus(ctx context.Context, client *xelon.Client, kubernetesClusterID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		kubernetesCluster, _, err := client.Kubernetes.Get(ctx, kubernetesClusterID)
		if err != nil {
			return nil, "", err
		}
		if kubernetesCluster == nil {
			return nil, "", fmt.Errorf("failed to kubernetes cluster with id: %s", kubernetesClusterID)
		}

		switch kubernetesCluster.Status {
		case "Ready":
			return kubernetesCluster, xksClusterStatusReady, nil
		case "Deleting", "Deleted", "Error":
			return nil, "", fmt.Errorf("kubernetes cluster %s entered terminal state %q while waiting for ready", kubernetesClusterID, kubernetesCluster.Status)
		default:
			return kubernetesCluster, xksClusterStatusProvisioning, nil
		}
	}
}

func WaitXKSClusterControlPlaneStatusHealthy(ctx context.Context, client *xelon.Client, kubernetesClusterID string, timeout time.Duration) error {
	stateConf := &retry.StateChangeConf{
		Pending:    []string{xksClusterStatusControlPlaneProvisioning},
		Target:     []string{xksClusterStatusControlPlaneHealthy},
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      5 * time.Second,
		Refresh:    statusXKSClusterControlPlaneStateStatus(ctx, client, kubernetesClusterID),
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("failed to wait for kubernetes cluster (%s) to become healthy: %w", kubernetesClusterID, err)
	}
	return nil
}

func statusXKSClusterControlPlaneStateStatus(ctx context.Context, client *xelon.Client, kubernetesClusterID string) retry.StateRefreshFunc {
	return func() (any, string, error) {
		kubernetesCluster, _, err := client.Kubernetes.Get(ctx, kubernetesClusterID)
		if err != nil {
			return nil, "", err
		}
		if kubernetesCluster == nil {
			return nil, "", fmt.Errorf("failed to kubernetes cluster with id: %s", kubernetesClusterID)
		}
		if kubernetesCluster.Health == nil {
			return nil, "", fmt.Errorf("failed to kubernetes cluster %s health data", kubernetesClusterID)
		}

		if kubernetesCluster.Health.Status == "healthy" {
			return kubernetesCluster, xksClusterStatusControlPlaneHealthy, nil
		}
		return kubernetesCluster, xksClusterStatusControlPlaneProvisioning, nil
	}
}
