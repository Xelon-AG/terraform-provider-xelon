package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDataSourceXelonFirewall(t *testing.T) {
	firewallName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceXelonFirewallConfig(firewallName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.xelon_firewall.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_firewall.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(firewallName),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_firewall.test",
						tfjsonpath.New("cloud_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_firewall.test",
						tfjsonpath.New("tenant_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_firewall.test",
						tfjsonpath.New("internal_ipv4_address"),
						knownvalue.NotNull(),
					),
				},
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

func testAccDataSourceXelonFirewallConfig(name string) string {
	return fmt.Sprintf(`
data "xelon_firewall" "test" {
  id = xelon_firewall.test.id
}

resource "xelon_firewall" "test" {
  cloud_id             = "e96db9d92ec7"
  internal_network_id  = "654871d16146"
  name                 = %[1]q
  tenant_id            = data.xelon_tenant.test.id
}

data "xelon_tenant" "test" {}
`, name)
}

const testAccDataSourceXelonFirewallConfigWithoutIDAndName = `
data "xelon_firewall" "test" {}
`
