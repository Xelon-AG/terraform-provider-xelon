package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDataSourceXelonFirewall(t *testing.T) {
	t.Skip()

	firewallName := acctest.RandomWithPrefix(accTestPrefix)
	stateChecks := []statecheck.StateCheck{
		statecheck.CompareValuePairs(
			"data.xelon_firewall.test", tfjsonpath.New("id"),
			"xelon_firewall.test", tfjsonpath.New("id"),
			compare.ValuesSame(),
		),
		statecheck.CompareValuePairs(
			"data.xelon_firewall.test", tfjsonpath.New("name"),
			"xelon_firewall.test", tfjsonpath.New("name"),
			compare.ValuesSame(),
		),
		statecheck.CompareValuePairs(
			"data.xelon_firewall.test", tfjsonpath.New("cloud_id"),
			"xelon_firewall.test", tfjsonpath.New("cloud_id"),
			compare.ValuesSame(),
		),
		statecheck.CompareValuePairs(
			"data.xelon_firewall.test", tfjsonpath.New("tenant_id"),
			"xelon_firewall.test", tfjsonpath.New("tenant_id"),
			compare.ValuesSame(),
		),
		statecheck.CompareValuePairs(
			"data.xelon_firewall.test", tfjsonpath.New("internal_ipv4_address"),
			"xelon_firewall.test", tfjsonpath.New("internal_ipv4_address"),
			compare.ValuesSame(),
		),
		statecheck.CompareValuePairs(
			"data.xelon_firewall.test", tfjsonpath.New("external_ipv4_address"),
			"xelon_firewall.test", tfjsonpath.New("external_ipv4_address"),
			compare.ValuesSame(),
		),
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            testAccDataSourceXelonFirewallConfigWithID(firewallName),
				ConfigStateChecks: stateChecks,
			},
			{
				Config:            testAccDataSourceXelonFirewallConfigWithName(firewallName),
				ConfigStateChecks: stateChecks,
			},
			{
				Config:            testAccDataSourceXelonFirewallConfigWithIDAndName(firewallName),
				ConfigStateChecks: stateChecks,
			},
		},
	})
}

func TestAccDataSourceXelonFirewall_expectError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceXelonFirewallConfigWithoutIDAndName,
				ExpectError: regexp.MustCompile(`The attribute "id" or "name" must be defined`),
			},
		},
	})
}

func testAccDataSourceXelonFirewallConfigWithID(name string) string {
	return fmt.Sprintf(`
data "xelon_firewall" "test" {
  id = xelon_firewall.test.id
}

resource "xelon_firewall" "test" {
  cloud_id            = "e96db9d92ec7"
  internal_network_id = "654871d16146"
  name                = %q
  tenant_id           = data.xelon_tenant.test.id
}

data "xelon_tenant" "test" {}
`, name)
}

func testAccDataSourceXelonFirewallConfigWithName(name string) string {
	return fmt.Sprintf(`
data "xelon_firewall" "test" {
  name = xelon_firewall.test.name
}

resource "xelon_firewall" "test" {
  cloud_id            = "e96db9d92ec7"
  internal_network_id = "654871d16146"
  name                = %q
  tenant_id           = data.xelon_tenant.test.id
}

data "xelon_tenant" "test" {}
`, name)
}

func testAccDataSourceXelonFirewallConfigWithIDAndName(name string) string {
	return fmt.Sprintf(`
data "xelon_firewall" "test" {
  id   = xelon_firewall.test.id
  name = xelon_firewall.test.name
}

resource "xelon_firewall" "test" {
  cloud_id            = "e96db9d92ec7"
  internal_network_id = "654871d16146"
  name                = %q
  tenant_id           = data.xelon_tenant.test.id
}

data "xelon_tenant" "test" {}
`, name)
}

const testAccDataSourceXelonFirewallConfigWithoutIDAndName = `
data "xelon_firewall" "test" {}
`
