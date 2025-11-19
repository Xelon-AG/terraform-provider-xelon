resource "xelon_load_balancer" "backend_ui" {
  cloud_id   = "<cloud-id>"
  name       = "backend UI load balancer"
  network_id = "<network-id>"
  tenant_id  = "<tenant-id>"
  type       = "layer4"
}
