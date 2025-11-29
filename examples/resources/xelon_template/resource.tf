resource "xelon_template" "base" {
  device_id = "<device-id>"
  name      = "base-debian-13.2"
  tenant_id = data.xelon_tenant.current.id
}

data "xelon_tenant" "current" {}
