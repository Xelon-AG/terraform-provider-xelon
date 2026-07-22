resource "xelon_object_storage_user" "example" {
  name          = "example-user"
  region        = "zh1"
  storage_limit = 500
}

resource "xelon_object_storage_bucket" "example_locked" {
  name                       = "example-locked-bucket"
  user_id                    = xelon_object_storage_user.example.id
  object_lock_enabled        = true
  object_lock_retention_days = 30
  versioning_enabled         = true
}
