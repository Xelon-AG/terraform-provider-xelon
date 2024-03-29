resource "xelon_network" "backend_lan" {
  cloud_id      = data.xelon_cloud.hcp.cloud_id
  dns_primary   = "8.8.8.8"
  dns_secondary = "8.8.4.4"
  gateway       = "10.0.0.1"
  name          = "LAN: backend"
  network       = "10.0.0.0"
  type          = "LAN"
}

data "xelon_cloud" "hcp" {
  name = "Main HCP Cloud"
}
