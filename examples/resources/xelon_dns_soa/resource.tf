resource "xelon_dns_zone" "public" {
  name = "mydomain.com"
}

resource "xelon_dns_soa" "public" {
  zone_id = xelon_dns_zone.public.id

  primary_nameserver = "ns1.xdns.cloud"
  admin_email        = "support@cloudns.net"
  refresh            = 7200
  retry              = 1800
  expire             = 1209600
  ttl                = 3600
}
