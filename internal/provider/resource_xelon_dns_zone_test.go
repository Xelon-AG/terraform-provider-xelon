package provider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func init() {
	resource.AddTestSweepers("xelon_dns_zone", &resource.Sweeper{
		Name: "xelon_dns_zone",
		F: func(region string) error {
			ctx := context.Background()
			client, err := sharedClient(region)
			if err != nil {
				return err
			}

			dnsZones, errf := client.Domains.AllZones(ctx, &xelon.ListOptions{PerPage: 100})
			for dnsZone := range dnsZones {
				if strings.HasPrefix(dnsZone.Name, accTestPrefix) {
					slog.Info("Deleting xelon_dns_zone", "name", dnsZone.Name, "id", dnsZone.ID)
					_, err := client.Domains.DeleteZone(ctx, dnsZone.ID)
					if err != nil {
						slog.Warn("Error deleting dns zone during sweep", "name", dnsZone.Name, "error", err)
					}
				}
			}
			if err := errf(); err != nil {
				return fmt.Errorf("getting dns zone list: %w", err)
			}

			return nil
		},
	})
}

func TestAccResourceXelonDNSZone(t *testing.T) {
	name := fmt.Sprintf("%s.xelon.cloud", acctest.RandomWithPrefix(accTestPrefix))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read
			{
				Config: testAccResourceXelonDNSZoneConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_dns_zone.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_dns_zone.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
				},
			},
		},
	})
}

func testAccResourceXelonDNSZoneConfig(name string) string {
	return fmt.Sprintf(`
resource "xelon_dns_zone" "test" {
  name = %[1]q
}
`, name)
}
