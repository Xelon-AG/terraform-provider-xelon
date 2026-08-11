package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/require"
)

func TestAccResourceXelonDeviceBackup(t *testing.T) {
	hostname := acctest.RandomWithPrefix(accTestPrefix)
	backupPlanIDs := testAccResourceXelonDeviceBackupPlanIDs(t)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Assign a backup plan and read back the authoritative assignment.
			{
				Config: testAccResourceXelonDeviceBackupConfig(hostname, backupPlanIDs[0]),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_device_backup.test",
						tfjsonpath.New("device_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_device_backup.test",
						tfjsonpath.New("backup_plan_id"),
						knownvalue.Int64Exact(int64(backupPlanIDs[0])),
					),
				},
			},
			// Import by device ID and verify Read reconstructs the assignment.
			{
				ResourceName: "xelon_device_backup.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["xelon_device_backup.test"]
					if !ok {
						return "", fmt.Errorf("not found: xelon_device_backup.test")
					}
					return rs.Primary.Attributes["device_id"], nil
				},
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "device_id",
			},
			// Change the assigned plan and verify the assignment updates in place.
			{
				Config: testAccResourceXelonDeviceBackupConfig(hostname, backupPlanIDs[1]),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_device_backup.test", plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_device_backup.test",
						tfjsonpath.New("backup_plan_id"),
						knownvalue.Int64Exact(int64(backupPlanIDs[1])),
					),
				},
			},
			// Remove the resource and verify backups are disabled while the device remains.
			{
				Config: testAccResourceXelonDeviceBackupDeviceOnlyConfig(hostname),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_device_backup.test", plancheck.ResourceActionDestroy),
					},
				},
				Check: testAccResourceXelonDeviceBackupIsDisabled("xelon_device.test"),
			},
			// Recreate the assignment for disappeared-resource coverage.
			{
				Config: testAccResourceXelonDeviceBackupConfig(hostname, backupPlanIDs[1]),
			},
			// Disable backups outside Terraform and verify refresh plans recreation.
			{
				Config:             testAccResourceXelonDeviceBackupConfig(hostname, backupPlanIDs[1]),
				Check:              testAccResourceXelonDeviceBackupDisable("xelon_device_backup.test"),
				ExpectNonEmptyPlan: true,
			},
			// Recreate the assignment after Read removed the disappeared resource from state.
			{
				Config: testAccResourceXelonDeviceBackupConfig(hostname, backupPlanIDs[1]),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("xelon_device_backup.test", plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func testAccResourceXelonDeviceBackupConfig(hostname string, backupPlanID int) string {
	return fmt.Sprintf(`
resource "xelon_device" "test" {
  cpu_core_count = 2
  disk_size      = 10
  display_name   = %[1]q
  hostname       = %[1]q
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

resource "xelon_device_backup" "test" {
  device_id      = xelon_device.test.id
  backup_plan_id = %[2]d
}

data "xelon_tenant" "test" {}

data "xelon_template" "test" {
  cloud_id    = "e96db9d92ec7"
  name        = "Debian 11"
  most_recent = true
}
`, hostname, backupPlanID)
}

func testAccResourceXelonDeviceBackupDeviceOnlyConfig(hostname string) string {
	return fmt.Sprintf(`
resource "xelon_device" "test" {
  cpu_core_count = 2
  disk_size      = 10
  display_name   = %[1]q
  hostname       = %[1]q
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
`, hostname)
}

func testAccResourceXelonDeviceBackupPlanIDs(t *testing.T) []int {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		return []int{1, 2}
	}

	testAccPreCheck(t)
	client, err := sharedClient("testacc")
	require.NoError(t, err)

	plans, _, err := client.Backups.ListPlans(context.Background(), "e96db9d92ec7")
	require.NoError(t, err)
	if len(plans) < 2 {
		t.Skip("xelon_device_backup acceptance test requires at least two backup plans in the device cloud")
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].ID < plans[j].ID
	})

	return []int{plans[0].ID, plans[1].ID}
}

func testAccResourceXelonDeviceBackupDisable(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		deviceID := rs.Primary.Attributes["device_id"]
		resp, err := client.Backups.DisableDeviceBackup(context.Background(), deviceID)
		if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
			return err
		}

		return nil
	}
}

func testAccResourceXelonDeviceBackupIsDisabled(deviceResourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[deviceResourceName]
		if !ok {
			return fmt.Errorf("not found: %s", deviceResourceName)
		}

		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		backupPlan, _, err := client.Backups.GetDevicePlan(context.Background(), rs.Primary.Attributes["id"])
		if err != nil {
			return err
		}
		if backupPlan != nil {
			return fmt.Errorf("device backup assignment still exists with backup plan ID %d", backupPlan.ID)
		}

		return nil
	}
}
