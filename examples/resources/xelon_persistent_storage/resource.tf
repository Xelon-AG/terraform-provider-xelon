resource "xelon_persistent_storage" "database" {
  cloud_id  = "<cloud-id>"
  device_id = "<device-id>"
  name      = "database-postgres-storage"
  size      = 50
  tenant_id = "<tenant-id>"
}
