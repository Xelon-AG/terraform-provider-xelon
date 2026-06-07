resource "xelon_kubernetes_cluster" "staging" {
  name      = "staging"
  cloud_id  = "<cloud-id>"
  tenant_id = "<tenant-id>"

  talos_version      = data.xelon_kubernetes_cluster_versions.staging.latest.talos_version
  kubernetes_version = data.xelon_kubernetes_cluster_versions.staging.latest.kubernetes_version

  control_plane = {
    cpu_core_count = 4
    disk_size      = 100
    memory         = 8
  }
}

data "xelon_kubernetes_cluster_versions" "staging" {
  cloud_id = "<cloud-id>"
}

# Run "terraform output -raw kubeconfig > ~/kubeconfig.dms"
# to persist kubeconfig for your new created cluster
output "kubeconfig" {
  value     = xelon_kubernetes_cluster.staging.kube_config_raw
  sensitive = true
}
