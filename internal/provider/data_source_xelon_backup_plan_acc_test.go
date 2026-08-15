package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const (
	testAccDataSourceXelonBackupPlanCloudID = "e96db9d92ec7"
	testAccDataSourceXelonBackupPlanID      = 20
	testAccDataSourceXelonBackupPlanName    = "Daily Backup, kept for 5 days"
)

func TestAccDataSourceXelonBackupPlan(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceXelonBackupPlanConfig(
					testAccDataSourceXelonBackupPlanCloudID,
					testAccDataSourceXelonBackupPlanName,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.xelon_backup_plan.test",
						tfjsonpath.New("cloud_id"),
						knownvalue.StringExact(testAccDataSourceXelonBackupPlanCloudID),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_backup_plan.test",
						tfjsonpath.New("id"),
						knownvalue.Int64Exact(testAccDataSourceXelonBackupPlanID),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_backup_plan.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(testAccDataSourceXelonBackupPlanName),
					),
				},
			},
		},
	})
}

func testAccDataSourceXelonBackupPlanConfig(cloudID, backupPlanName string) string {
	return fmt.Sprintf(`
data "xelon_backup_plan" "test" {
  cloud_id = %q
  name     = %q
}
`, cloudID, backupPlanName)
}
