resource "xelon_object_storage_user" "example" {
  name          = "example-user"
  region        = "zh1"
  storage_limit = 500
}

resource "xelon_object_storage_bucket" "example_versioned" {
  name               = "example-versioned-bucket"
  user_id            = xelon_object_storage_user.example.id
  versioning_enabled = true
}
