resource "xelon_kubernetes_node_pool" "default" {
  kubernetes_cluster_id = "<kubernetes-cluster-id>"
  name                  = "default"
}
