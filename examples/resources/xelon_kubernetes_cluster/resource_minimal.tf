resource "xelon_kubernetes_cluster" "staging" {
  name      = "staging"
  cloud_id  = "<cloud-id>"
  tenant_id = "<tenant-id>"

  talos_version      = data.xelon_kubernetes_cluster_versions.staging.latest.talos_version
  kubernetes_version = data.xelon_kubernetes_cluster_versions.staging.latest.kubernetes_version
}

data "xelon_kubernetes_cluster_versions" "staging" {
  cloud_id = "<cloud-id>"
}
