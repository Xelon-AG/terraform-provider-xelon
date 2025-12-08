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

func TestAccDataSourceXelonISO(t *testing.T) {
	isoName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceXelonISOConfig(isoName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.xelon_iso.test",
						tfjsonpath.New("cloud_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_iso.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_iso.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(isoName),
					),
				},
			},
		},
	})
}

func TestAccDataSourceXelonISO_expectError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceXelonISOConfigWithoutIDAndName,
				ExpectError: regexp.MustCompile(`The attribute "id" or "name" must be defined`),
			},
		},
	})
}

func testAccDataSourceXelonISOConfig(name string) string {
	return fmt.Sprintf(`
data "xelon_iso" "test" {
  id = xelon_iso.test.id
}

resource "xelon_iso" "test" {
  category_id = 2
  cloud_id    = data.xelon_cloud.test.id
  name = %[1]q
  url  = "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-13.2.0-amd64-netinst.iso"
}

data "xelon_cloud" "test" {
  # cloud id from test environment, change if updated
  id = "c9ab2f0fcfde"
}
`, name)
}

const testAccDataSourceXelonISOConfigWithoutIDAndName = `
data "xelon_iso" "test" {}
`
