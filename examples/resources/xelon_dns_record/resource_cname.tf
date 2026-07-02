resource "xelon_dns_zone" "public" {
  name = "mydomain.com"
}

resource "xelon_dns_record" "app" {
  zone_id = xelon_dns_zone.public.id

  name    = "app"
  type    = "CNAME"
  content = "target.example.net"
  ttl     = 3600
}
