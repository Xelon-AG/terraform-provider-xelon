data "xelon_backup_plan" "daily" {
  cloud_id = data.xelon_cloud.example.id
  name     = "Daily Backup, kept for 5 days"
}
