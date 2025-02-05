resource "xelon_device" "server" {
  cpu_core_count = 2
  disk_size      = 10
  display_name   = "server"
  hostname       = "server"
  memory         = 2
  password       = "<generated-secure-password>"
  swap_disk_size = 1
  template_id    = "<template-id>"
  tenant_id      = "<tenant-id>"

  networks = [
    {
      connected          = true
      id                 = "<network-id>"
      ipv4_address       = "10.0.0.155"
      nic_controller_key = 100
      nic_key            = 4000
      nic_unit_number    = 7
    }
  ]
}
