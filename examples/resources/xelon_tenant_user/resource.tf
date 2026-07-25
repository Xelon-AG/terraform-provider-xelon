resource "xelon_tenant_user" "example" {
  tenant_id  = data.xelon_tenant.example.id
  email      = "john.doe@example.com"
  first_name = "John"
  last_name  = "Doe"
  password   = var.tenant_user_password

  roles       = ["hq_root_admin"]
  permissions = ["allow_view_virtual_machines"]
}

data "xelon_tenant" "example" {}

variable "tenant_user_password" {
  type      = string
  sensitive = true
}
