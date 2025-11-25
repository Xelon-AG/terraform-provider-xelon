resource "xelon_firewall" "frontend" {
  cloud_id            = "<cloud-id>"
  internal_network_id = "<internal-network-id>"
  name                = "frontend firewall"
  tenant_id           = "<tenant-id>"
}
