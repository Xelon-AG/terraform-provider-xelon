data "xelon_kubernetes_cluster" "example" {
  kubernetes_cluster_id = "<kubernetes-cluster-id>"
}

output "kubeconfig" {
  value     = data.xelon_kubernetes_cluster.example.kubeconfig_raw
  sensitive = true
}
