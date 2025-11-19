resource "xelon_load_balancer_forwarding_rule" "web" {
  load_balancer_id = "<load-balancer-id>"

  ipv4_addresses = ["10.0.0.50"]
  from_port      = 443
  to_port        = 8080
}
