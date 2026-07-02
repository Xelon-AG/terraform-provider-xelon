resource "xelon_dns_zone" "public" {
  name = "mydomain.com"
}

resource "xelon_dns_record" "www" {
  zone_id = xelon_dns_zone.public.id

  name    = "www"
  type    = "A"
  content = "203.0.113.10"
  ttl     = 3600
}
