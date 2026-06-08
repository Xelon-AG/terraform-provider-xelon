data "xelon_kubernetes_cluster_versions" "hcp" {
  cloud_id = data.xelon_cloud.hcp.id
}

data "xelon_cloud" "hcp" {
  name = "Main HCP Cloud"
}
