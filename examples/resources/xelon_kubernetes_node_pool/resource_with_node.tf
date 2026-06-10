resource "xelon_kubernetes_node_pool" "main" {
  kubernetes_cluster_id = "<kubernetes-cluster-id>"
  name                  = "main pool"

  node_count = 6

  cpu_core_count = 8
  disk_size      = 100
  memory         = 8
}
