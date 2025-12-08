package provider

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
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
	resource.AddTestSweepers("xelon_iso", &resource.Sweeper{
		Name: "xelon_iso",
		F:    testSweepISOs,
	})
}

func testSweepISOs(region string) error {
	ctx := context.Background()
	client, err := sharedClient(region)
	if err != nil {
		return err
	}

	isos, _, err := client.ISOs.List(ctx, &xelon.ISOListOptions{ListOptions: xelon.ListOptions{PerPage: 100}})
	if err != nil {
		return fmt.Errorf("getting ISO list: %s", err)
	}

	for _, iso := range isos {
		if strings.HasPrefix(iso.Name, accTestPrefix) {
			slog.Info("Deleting xelon_iso", "name", iso.Name, "id", iso.ID)
			_, err := client.ISOs.Delete(ctx, iso.ID)
			if err != nil {
				slog.Warn("Error deleting ISO during sweep", "name", iso.Name, "error", err)
			}
		}
	}

	return nil
}

func TestAccResourceXelonISO(t *testing.T) {
	isoName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	isoDescription := fmt.Sprintf("%s-description", accTestPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read
			{
				Config: testAccResourceXelonISOConfig(isoName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_iso.test",
						tfjsonpath.New("active"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_iso.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"xelon_iso.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// update and read
			{
				Config: testAccResourceXelonISOConfigWithDescription(isoName, isoDescription),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_iso.test",
						tfjsonpath.New("active"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_iso.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(isoDescription),
					),
					statecheck.ExpectKnownValue(
						"xelon_iso.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccResourceXelonISO_expectError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccResourceXelonISOConfigWithoutName,
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
			{
				Config:      testAccResourceXelonISOConfigWithoutCategoryID,
				ExpectError: regexp.MustCompile(`The argument "category_id" is required`),
			},
			{
				Config:      testAccResourceXelonISOConfigWithoutURL,
				ExpectError: regexp.MustCompile(`The argument "url" is required`),
			},
		},
	})
}

func testAccResourceXelonISOConfig(name string) string {
	return fmt.Sprintf(`
resource "xelon_iso" "test" {
  category_id = 2
  cloud_id    = data.xelon_cloud.test.id
  name        = %[1]q
  url         = "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-13.2.0-amd64-netinst.iso"
}

data "xelon_cloud" "test" {
  # cloud id from test environment, change if updated
  id = "c9ab2f0fcfde"
}
`, name)
}

func testAccResourceXelonISOConfigWithDescription(name, description string) string {
	return fmt.Sprintf(`
resource "xelon_iso" "test" {
  category_id = 2
  cloud_id    = data.xelon_cloud.test.id
  description = %[2]q
  name        = %[1]q
  url         = "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-13.2.0-amd64-netinst.iso"
}

data "xelon_cloud" "test" {
  # cloud id from test environment, change if updated
  id = "c9ab2f0fcfde"
}
`, name, description)
}

const testAccResourceXelonISOConfigWithoutName = `
resource "xelon_iso" "test" {
  category_id = 2
  cloud_id    = "random-id"
  url         = "random-url"
}
`

const testAccResourceXelonISOConfigWithoutCategoryID = `
resource "xelon_iso" "test" {
  cloud_id = "random-id"
  name     = "random-name"
  url      = "random-url"
}
`

const testAccResourceXelonISOConfigWithoutURL = `
resource "xelon_iso" "test" {
  category_id = 2
  cloud_id    = "random-id"
  name        = "random-name"
}
`
