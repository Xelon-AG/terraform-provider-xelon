resource "xelon_device" "server" {
  cloud_id       = data.xelon_cloud.hcp.cloud_id
  cpu_core_count = 2
  disk_size      = 10
  display_name   = "server"
  hostname       = "server"
  memory         = 2
  password       = "<generated-secure-password>"
  swap_disk_size = 1
  template_id    = 10000

  // find correct network values from template creation info
  network {
    id                 = 100
    ipv4_address_id    = 100
    nic_controller_key = 100
    nic_key            = 100
    nic_number         = 100
    nic_unit           = 100
  }
}

data "xelon_cloud" "hcp" {
  name = "Main HCP Cloud"
}
