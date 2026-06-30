resource "xelon_object_storage_access_key" "test" {
  user_id = xelon_object_storage_user.test.id
}

resource "xelon_object_storage_user" "test" {
  name          = "testuser"
  region        = "zh1"
  storage_limit = 500
}
