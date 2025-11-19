data "xelon_ssh_key" "admin_for_device" {
  name = "admin-key"
}

data "xelon_template" "ubuntu_for_ssh_example" {
  name = "Ubuntu 22.04 LTS"
}

data "xelon_tenant" "current_for_ssh_example" {}

data "xelon_network" "lan_for_ssh_example" {
  name = "LAN"
}

resource "xelon_device" "web_with_ssh" {
  template_id      = data.xelon_template.ubuntu_for_ssh_example.id
  tenant_id        = data.xelon_tenant.current_for_ssh_example.id
  ssh_key_id       = data.xelon_ssh_key.admin_for_device.id
  display_name     = "web-server"
  hostname         = "web01"
  cpu_core_count   = 2
  memory           = 4
  disk_size        = 40
  swap_disk_size   = 2
  password         = "SecurePassword123!"  # Should use a variable in production

  networks = [
    {
      id        = data.xelon_network.lan_for_ssh_example.id
      connected = true
    }
  ]
}
