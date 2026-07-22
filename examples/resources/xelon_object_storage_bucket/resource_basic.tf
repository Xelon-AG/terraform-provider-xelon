resource "xelon_object_storage_user" "example" {
  name          = "example-user"
  region        = "zh1"
  storage_limit = 500
}

resource "xelon_object_storage_bucket" "example" {
  name    = "example-bucket"
  user_id = xelon_object_storage_user.example.id
}
