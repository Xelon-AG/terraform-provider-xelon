resource "xelon_device_backup" "example" {
  device_id      = xelon_device.example.id
  backup_plan_id = 17
}
