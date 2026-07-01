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
	resource.AddTestSweepers("xelon_device", &resource.Sweeper{
		Name: "xelon_device",
		F: func(region string) error {
			ctx := context.Background()
			client, err := sharedClient(region)
			if err != nil {
				return err
			}

			devices, errf := client.Devices.All(ctx, &xelon.ListOptions{PerPage: 100})
			for device := range devices {
				if strings.HasPrefix(device.DisplayName, accTestPrefix) {
					slog.Info("Deleting xelon_device", "name", device.DisplayName, "id", device.ID)
					_, err := client.Devices.Delete(ctx, device.ID)
					if err != nil {
						slog.Warn("Error deleting device during sweep", "name", device.DisplayName, "error", err)
					}
				}
			}
			if err := errf(); err != nil {
				return fmt.Errorf("getting device list: %w", err)
			}

			return nil
		},
	})
}

func TestAccResourceXelonDevice(t *testing.T) {
	hostname := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	displayName := hostname
	displayNameUpdated := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read
			{
				Config: testAccResourceXelonDeviceConfig(displayName, hostname, 10),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("cpu_core_count"),
						knownvalue.Int64Exact(2),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("disk_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("disk_size"),
						knownvalue.Int64Exact(10),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact(displayName),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("enable_monitoring"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("hostname"),
						knownvalue.StringExact(hostname),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("memory"),
						knownvalue.Int64Exact(2),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("swap_disk_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("swap_disk_size"),
						knownvalue.Int64Exact(1),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("tenant_id"),
						knownvalue.NotNull(),
					),
				},
			},
			// update and read
			{
				Config: testAccResourceXelonDeviceConfig(displayNameUpdated, hostname, 15),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("cpu_core_count"),
						knownvalue.Int64Exact(2),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("disk_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("disk_size"),
						knownvalue.Int64Exact(15),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact(displayNameUpdated),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("enable_monitoring"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("hostname"),
						knownvalue.StringExact(hostname),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("memory"),
						knownvalue.Int64Exact(2),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("swap_disk_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("swap_disk_size"),
						knownvalue.Int64Exact(1),
					),
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("tenant_id"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccResourceXelonDevice_autoAssignedIPAddress(t *testing.T) {
	hostname := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	displayName := hostname

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create a cloud-init device selecting only the network, so Xelon
			// auto-assigns the IP address, and read it back into state
			{
				Config: testAccResourceXelonDeviceAutoAssignedIPAddressConfig(displayName, hostname),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					// the network is created without an explicit ipv4_address, so this
					// asserts the auto-assigned address is read back into state as a
					// known, non-null value (e.g. to feed load balancer forwarding rules).
					statecheck.ExpectKnownValue(
						"xelon_device.test",
						tfjsonpath.New("networks").AtSliceIndex(0).AtMapKey("ipv4_address"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccResourceXelonDeviceConfig(displayName, hostname string, diskSize int) string {
	return fmt.Sprintf(`
resource "xelon_device" "test" {
  cpu_core_count = 2
  disk_size      = %[3]d
  display_name   = %[1]q
  hostname       = %[2]q
  memory         = 2
  password       = "J78q3H"
  swap_disk_size = 1
  template_id    = data.xelon_template.test.id
  tenant_id      = data.xelon_tenant.test.id

  networks = [
    {
      connected = true
      id        = "654871d16146"
    }
  ]
}

data "xelon_tenant" "test" {}

data "xelon_template" "test" {
  cloud_id    = "e96db9d92ec7"
  name        = "Debian 11"
  most_recent = true
}
`, displayName, hostname, diskSize)
}

func testAccResourceXelonDeviceAutoAssignedIPAddressConfig(displayName, hostname string) string {
	return fmt.Sprintf(`
resource "xelon_device" "test" {
  cpu_core_count = 2
  disk_size      = 10
  display_name   = %[1]q
  hostname       = %[2]q
  memory         = 2
  template_id    = data.xelon_template.test.id
  tenant_id      = data.xelon_tenant.test.id
  user_data      = <<-EOT
#cloud-config
hostname: %[2]s
EOT

  networks = [
    {
      connected = true
      id        = "654871d16146"
    }
  ]
}

data "xelon_tenant" "test" {}

data "xelon_template" "test" {
  cloud_id    = "e96db9d92ec7"
  name        = "Debian 11"
  most_recent = true
}
`, displayName, hostname)
}
