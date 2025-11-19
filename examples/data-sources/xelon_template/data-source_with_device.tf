data "xelon_template" "ubuntu_for_device" {
  name = "Ubuntu 22.04 LTS"
}

data "xelon_tenant" "current" {}

data "xelon_network" "lan" {
  name = "LAN"
}

resource "xelon_device" "web" {
  template_id      = data.xelon_template.ubuntu_for_device.id
  tenant_id        = data.xelon_tenant.current.id
  display_name     = "web-server"
  hostname         = "web01"
  cpu_core_count   = 2
  memory           = 4
  disk_size        = 40
  swap_disk_size   = 2
  password         = "SecurePassword123!"  # Should use a variable in production

  networks = [
    {
      id        = data.xelon_network.lan.id
      connected = true
    }
  ]
}
