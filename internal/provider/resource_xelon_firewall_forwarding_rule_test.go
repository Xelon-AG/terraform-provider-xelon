package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func init() {
	resource.AddTestSweepers("xelon_firewall_forwarding_rule", &resource.Sweeper{
		Name: "xelon_firewall_forwarding_rule",
		F:    testSweepFirewallForwardingRules,
		Dependencies: []string{
			"xelon_firewall",
		},
	})
}

func testSweepFirewallForwardingRules(region string) error {
	ctx := context.Background()
	client, err := sharedClient(region)
	if err != nil {
		return err
	}

	firewalls, _, err := client.Firewalls.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("getting firewall list: %s", err)
	}

	for _, firewall := range firewalls {
		if !strings.HasPrefix(firewall.Name, accTestPrefix) {
			continue
		}

		// Get firewall with its forwarding rules
		firewallDetails, _, err := client.Firewalls.Get(ctx, firewall.ID)
		if err != nil {
			slog.Warn("Error getting firewall during sweep", "firewall_id", firewall.ID, "error", err)
			continue
		}

		// Delete forwarding rules for this firewall
		for _, rule := range firewallDetails.ForwardingRules {
			slog.Info("Deleting xelon_firewall_forwarding_rule", "firewall_id", firewall.ID, "rule_id", rule.ID)
			_, err := client.Firewalls.DeleteForwardingRule(ctx, firewall.ID, rule.ID)
			if err != nil {
				slog.Warn("Error deleting firewall forwarding rule during sweep", "firewall_id", firewall.ID, "rule_id", rule.ID, "error", err)
			}
		}
	}

	return nil
}

// TestAccResourceXelonFirewallForwardingRule_InboundDestinationIPs tests that destination IP addresses
// are correctly preserved in state for inbound rules (Issue #1 - Critical Bug)
func TestAccResourceXelonFirewallForwardingRule_InboundDestinationIPs(t *testing.T) {
	var forwardingRule xelon.FirewallForwardingRule
	firewallName := fmt.Sprintf("%s-fw-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckFirewallForwardingRuleDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccXelonFirewallForwardingRuleInbound(firewallName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFirewallForwardingRuleExists("xelon_firewall_forwarding_rule.test", &forwardingRule),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "type", "inbound"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "protocol", "tcp"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "from_port", "80"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "to_port", "8080"),
					// Critical: Check destination IP addresses are preserved (Bug #1)
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.#", "1"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.*", "10.0.0.10"),
					// Check source IP addresses
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.#", "2"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.*", "0.0.0.0/0"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.*", "192.168.1.0/24"),
				),
			},
			// Test refresh to ensure state is preserved after read
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					// Critical: Destination IPs must still be in state after refresh
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.#", "1"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.*", "10.0.0.10"),
					// Source IPs must still be in state
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.#", "2"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.*", "0.0.0.0/0"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.*", "192.168.1.0/24"),
				),
			},
		},
	})
}

// TestAccResourceXelonFirewallForwardingRule_OutboundSourceIPs tests that source IP addresses
// are correctly preserved in state for outbound rules
func TestAccResourceXelonFirewallForwardingRule_OutboundSourceIPs(t *testing.T) {
	var forwardingRule xelon.FirewallForwardingRule
	firewallName := fmt.Sprintf("%s-fw-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckFirewallForwardingRuleDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccXelonFirewallForwardingRuleOutbound(firewallName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFirewallForwardingRuleExists("xelon_firewall_forwarding_rule.test", &forwardingRule),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "type", "outbound"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "protocol", "tcp"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "from_port", "3306"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "to_port", "3306"),
					// Check source IP addresses (single IP for outbound)
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.#", "1"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.*", "10.0.0.20"),
					// Check destination IP addresses (multiple for outbound)
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.#", "2"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.*", "8.8.8.8"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.*", "1.1.1.1"),
				),
			},
			// Test refresh
			{
				RefreshState: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.#", "1"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.*", "10.0.0.20"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.#", "2"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.*", "8.8.8.8"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.*", "1.1.1.1"),
				),
			},
		},
	})
}

// TestAccResourceXelonFirewallForwardingRule_Update tests that IP addresses are preserved
// after updating the rule (Bug #1 also exists in Update method)
func TestAccResourceXelonFirewallForwardingRule_Update(t *testing.T) {
	var forwardingRule xelon.FirewallForwardingRule
	firewallName := fmt.Sprintf("%s-fw-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckFirewallForwardingRuleDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccXelonFirewallForwardingRuleInbound(firewallName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFirewallForwardingRuleExists("xelon_firewall_forwarding_rule.test", &forwardingRule),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "from_port", "80"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "to_port", "8080"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.#", "1"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.*", "10.0.0.10"),
				),
			},
			{
				Config: testAccXelonFirewallForwardingRuleInboundUpdated(firewallName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFirewallForwardingRuleExists("xelon_firewall_forwarding_rule.test", &forwardingRule),
					// Ports changed
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "from_port", "443"),
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "to_port", "8443"),
					// Critical: Destination IPs must be preserved after update
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.#", "1"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "destination_ipv4_addresses.*", "10.0.0.10"),
					// Source IPs updated
					resource.TestCheckResourceAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.#", "1"),
					resource.TestCheckTypeSetElemAttr("xelon_firewall_forwarding_rule.test", "source_ipv4_addresses.*", "0.0.0.0/0"),
				),
			},
		},
	})
}

func testAccCheckFirewallForwardingRuleDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := sharedClient("testacc")
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "xelon_firewall_forwarding_rule" {
			continue
		}

		firewallID := rs.Primary.Attributes["firewall_id"]
		ruleID := rs.Primary.ID

		// Get firewall to check its forwarding rules
		firewall, _, err := client.Firewalls.Get(ctx, firewallID)
		if err != nil {
			// Firewall might be deleted, which is fine
			continue
		}

		// Check if the rule still exists
		for _, rule := range firewall.ForwardingRules {
			if rule.ID == ruleID {
				return fmt.Errorf("firewall forwarding rule (%s) still exists", ruleID)
			}
		}
	}

	return nil
}

func testAccCheckFirewallForwardingRuleExists(n string, forwardingRule *xelon.FirewallForwardingRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("no firewall forwarding rule ID set")
		}

		firewallID := rs.Primary.Attributes["firewall_id"]
		if firewallID == "" {
			return errors.New("no firewall ID set")
		}

		ctx := context.Background()
		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		// Get firewall to access its forwarding rules
		firewall, _, err := client.Firewalls.Get(ctx, firewallID)
		if err != nil {
			return fmt.Errorf("could not get firewall: %s", err)
		}

		// Find the specific forwarding rule
		for _, rule := range firewall.ForwardingRules {
			if rule.ID == rs.Primary.ID {
				*forwardingRule = rule
				return nil
			}
		}

		return fmt.Errorf("could not find firewall forwarding rule with ID: %s", rs.Primary.ID)
	}
}

// Test configurations

func testAccXelonFirewallForwardingRuleInbound(firewallName string) string {
	return fmt.Sprintf(`
data "xelon_tenant" "test" {}

data "xelon_cloud" "test" {
  name = "10. Devel 01 V"
}

resource "xelon_network" "internal" {
  cloud_id         = data.xelon_cloud.test.id
  tenant_id        = data.xelon_tenant.test.id
  name             = "%s-internal"
  type             = "LAN"
  network          = "10.0.0.0"
  subnet_size      = 24
  dns_primary      = "8.8.8.8"
  dns_secondary    = "8.8.4.4"
  gateway          = "10.0.0.1"
  network_speed    = 1000
}

resource "xelon_network" "external" {
  cloud_id      = data.xelon_cloud.test.id
  tenant_id     = data.xelon_tenant.test.id
  name          = "%s-external"
  type          = "WAN"
  subnet_size   = 29
  network_speed = 1000
}

resource "xelon_firewall" "test" {
  cloud_id             = data.xelon_cloud.test.id
  internal_network_id  = xelon_network.internal.id
  external_network_id  = xelon_network.external.id
  name                 = "%s"
  tenant_id            = data.xelon_tenant.test.id
}

resource "xelon_firewall_forwarding_rule" "test" {
  firewall_id                = xelon_firewall.test.id
  type                       = "inbound"
  protocol                   = "tcp"
  from_port                  = 80
  to_port                    = 8080
  source_ipv4_addresses      = ["0.0.0.0/0", "192.168.1.0/24"]
  destination_ipv4_addresses = ["10.0.0.10"]
}
`, firewallName, firewallName, firewallName)
}

func testAccXelonFirewallForwardingRuleInboundUpdated(firewallName string) string {
	return fmt.Sprintf(`
data "xelon_tenant" "test" {}

data "xelon_cloud" "test" {
  name = "10. Devel 01 V"
}

resource "xelon_network" "internal" {
  cloud_id         = data.xelon_cloud.test.id
  tenant_id        = data.xelon_tenant.test.id
  name             = "%s-internal"
  type             = "LAN"
  network          = "10.0.0.0"
  subnet_size      = 24
  dns_primary      = "8.8.8.8"
  dns_secondary    = "8.8.4.4"
  gateway          = "10.0.0.1"
  network_speed    = 1000
}

resource "xelon_network" "external" {
  cloud_id      = data.xelon_cloud.test.id
  tenant_id     = data.xelon_tenant.test.id
  name          = "%s-external"
  type          = "WAN"
  subnet_size   = 29
  network_speed = 1000
}

resource "xelon_firewall" "test" {
  cloud_id             = data.xelon_cloud.test.id
  internal_network_id  = xelon_network.internal.id
  external_network_id  = xelon_network.external.id
  name                 = "%s"
  tenant_id            = data.xelon_tenant.test.id
}

resource "xelon_firewall_forwarding_rule" "test" {
  firewall_id                = xelon_firewall.test.id
  type                       = "inbound"
  protocol                   = "tcp"
  from_port                  = 443
  to_port                    = 8443
  source_ipv4_addresses      = ["0.0.0.0/0"]
  destination_ipv4_addresses = ["10.0.0.10"]
}
`, firewallName, firewallName, firewallName)
}

func testAccXelonFirewallForwardingRuleOutbound(firewallName string) string {
	return fmt.Sprintf(`
data "xelon_tenant" "test" {}

data "xelon_cloud" "test" {
  name = "10. Devel 01 V"
}

resource "xelon_network" "internal" {
  cloud_id         = data.xelon_cloud.test.id
  tenant_id        = data.xelon_tenant.test.id
  name             = "%s-internal"
  type             = "LAN"
  network          = "10.0.0.0"
  subnet_size      = 24
  dns_primary      = "8.8.8.8"
  dns_secondary    = "8.8.4.4"
  gateway          = "10.0.0.1"
  network_speed    = 1000
}

resource "xelon_network" "external" {
  cloud_id      = data.xelon_cloud.test.id
  tenant_id     = data.xelon_tenant.test.id
  name          = "%s-external"
  type          = "WAN"
  subnet_size   = 29
  network_speed = 1000
}

resource "xelon_firewall" "test" {
  cloud_id             = data.xelon_cloud.test.id
  internal_network_id  = xelon_network.internal.id
  external_network_id  = xelon_network.external.id
  name                 = "%s"
  tenant_id            = data.xelon_tenant.test.id
}

resource "xelon_firewall_forwarding_rule" "test" {
  firewall_id                = xelon_firewall.test.id
  type                       = "outbound"
  protocol                   = "tcp"
  from_port                  = 3306
  to_port                    = 3306
  source_ipv4_addresses      = ["10.0.0.20"]
  destination_ipv4_addresses = ["8.8.8.8", "1.1.1.1"]
}
`, firewallName, firewallName, firewallName)
}
