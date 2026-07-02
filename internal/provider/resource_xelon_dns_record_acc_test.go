package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccResourceXelonDNSRecord(t *testing.T) {
	dnsZoneName := fmt.Sprintf("%s.xelon.cloud", acctest.RandomWithPrefix(accTestPrefix))
	content := "203.0.113.10"
	contentUpdated := "203.0.113.20"
	ttl := int64(1800)
	ttlUpdated := int64(3600)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read
			{
				Config: testAccResourceXelonDNSRecordConfig(dnsZoneName, "www", content, ttl),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("content"),
						knownvalue.StringExact(content),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("www"),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("record_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(ttl),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("zone_id"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "xelon_dns_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// update and read
			{
				Config: testAccResourceXelonDNSRecordConfig(dnsZoneName, "www", contentUpdated, ttlUpdated),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("content"),
						knownvalue.StringExact(contentUpdated),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("www"),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("record_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("ttl"),
						knownvalue.Int64Exact(ttlUpdated),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("type"),
						knownvalue.StringExact("A"),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_record.test",
						tfjsonpath.New("zone_id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccResourceXelonDNSRecordConfig(dnsZoneName, name, content string, ttl int64) string {
	return fmt.Sprintf(`
resource "xelon_dns_record" "test" {
  content = %[3]q
  name    = %[2]q
  ttl     = %[4]d
  type    = "A"
  zone_id = xelon_dns_zone.test.id
}

resource "xelon_dns_zone" "test" {
  name = %[1]q
}
`, dnsZoneName, name, content, ttl)
}
