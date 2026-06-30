resource "xelon_object_storage_user" "test" {
  name          = "testuser"
  region        = "zh1"
  storage_limit = 500
}
