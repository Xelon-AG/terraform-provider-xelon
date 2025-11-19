resource "xelon_firewall_forwarding_rule" "web" {
  firewall_id = "<firewall-id>"

  protocol = "tcp"
  type     = "inbound"

  destination_ipv4_addresses = ["10.0.0.50"]
  source_ipv4_addresses      = ["0.0.0.0/0"]
  from_port                  = 443
  to_port                    = 8080
}
