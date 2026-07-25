package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccResourceXelonDNSSOA(t *testing.T) {
	dnsZoneName := fmt.Sprintf("%s.xelon.cloud", acctest.RandomWithPrefix(accTestPrefix))

	initialIDMatchesZoneID := statecheck.CompareValue(compare.ValuesSame())
	initialIDMatchesZoneID.AddStateValue("xelon_dns_soa.test", tfjsonpath.New("id"))
	initialIDMatchesZoneID.AddStateValue("xelon_dns_soa.test", tfjsonpath.New("zone_id"))

	updatedIDMatchesZoneID := statecheck.CompareValue(compare.ValuesSame())
	updatedIDMatchesZoneID.AddStateValue("xelon_dns_soa.test", tfjsonpath.New("id"))
	updatedIDMatchesZoneID.AddStateValue("xelon_dns_soa.test", tfjsonpath.New("zone_id"))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// configure existing SOA settings and read back authoritative state
			{
				Config: testAccResourceXelonDNSSOAConfig(
					dnsZoneName,
					"ns1.xdns.cloud",
					"support@cloudns.net",
					7200,
					1800,
					1209600,
					3600,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("admin_email"),
						knownvalue.StringExact("support@cloudns.net"),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("expire"),
						knownvalue.Int64Exact(1209600),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("primary_nameserver"),
						knownvalue.StringExact("ns1.xdns.cloud"),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("refresh"),
						knownvalue.Int64Exact(7200),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("retry"),
						knownvalue.Int64Exact(1800),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(3600),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("zone_id"),
						knownvalue.NotNull(),
					),
					initialIDMatchesZoneID,
				},
			},
			// update mutable SOA settings in place and read back authoritative state
			{
				Config: testAccResourceXelonDNSSOAConfig(
					dnsZoneName,
					"ns2.xdns.cloud",
					"support_updated@cloudns.net",
					14400,
					3600,
					2419200,
					7200,
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_dns_soa.test", plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("admin_email"),
						knownvalue.StringExact("support_updated@cloudns.net"),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("expire"),
						knownvalue.Int64Exact(2419200),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("primary_nameserver"),
						knownvalue.StringExact("ns2.xdns.cloud"),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("refresh"),
						knownvalue.Int64Exact(14400),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("retry"),
						knownvalue.Int64Exact(3600),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(7200),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_soa.test",
						tfjsonpath.New("zone_id"),
						knownvalue.NotNull(),
					),
					updatedIDMatchesZoneID,
				},
			},
		},
	})
}

func testAccResourceXelonDNSSOAConfig(dnsZoneName, primaryNameserver, adminEmail string, refresh, retry, expire, ttl int64) string {
	return fmt.Sprintf(`
resource "xelon_dns_zone" "test" {
  name = %[1]q
}

resource "xelon_dns_soa" "test" {
  zone_id = xelon_dns_zone.test.id

  primary_nameserver = %[2]q
  admin_email        = %[3]q
  refresh            = %[4]d
  retry              = %[5]d
  expire             = %[6]d
  ttl                = %[7]d
}
`, dnsZoneName, primaryNameserver, adminEmail, refresh, retry, expire, ttl)
}
