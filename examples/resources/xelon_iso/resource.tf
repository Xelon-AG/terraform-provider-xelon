resource "xelon_iso" "debian" {
  category_id = 2
  cloud_id    = "<cloud-id>"
  description = "debian image 13.2.0"
  name        = "debian-13.2"
  url         = "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-13.2.0-amd64-netinst.iso"
}
