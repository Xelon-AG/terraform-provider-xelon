resource "xelon_persistent_storage" "backup" {
  cloud_id = data.xelon_cloud.hcp.cloud_id
  name     = "backup"
  size     = 50
}

data "xelon_cloud" "hcp" {
  name = "Main HCP Cloud"
}
