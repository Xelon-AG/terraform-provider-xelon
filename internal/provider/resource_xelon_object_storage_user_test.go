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
	resource.AddTestSweepers("xelon_object_storage_user", &resource.Sweeper{
		Name: "xelon_object_storage_user",
		F: func(region string) error {
			ctx := context.Background()
			client, err := sharedClient(region)
			if err != nil {
				return err
			}

			users, errf := client.ObjectStorages.AllUsers(ctx, &xelon.ListOptions{PerPage: 100})
			for user := range users {
				if strings.HasPrefix(user.Name, accTestPrefix) {
					slog.Info("Deleting xelon_object_storage_user", "name", user.Name, "id", user.ID)
					_, err := client.ObjectStorages.DeleteUser(ctx, user.ID)
					if err != nil {
						slog.Warn("Error deleting object storage user during sweep", "name", user.Name, "error", err)
					}
				}
			}
			if err := errf(); err != nil {
				return fmt.Errorf("getting object storage user list: %w", err)
			}
			return nil
		},
	})
}

func TestAccResourceXelonObjectStorageUser(t *testing.T) {
	name := acctest.RandomWithPrefix(accTestPrefix)
	nameUpdated := acctest.RandomWithPrefix(accTestPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read
			{
				Config: testAccResourceXelonObjectStorageUserConfig(name, 100),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(name),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("region"),
						knownvalue.StringExact("zh1"),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("s3_endpoints"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("storage_limit"),
						knownvalue.Int64Exact(100),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("tenant_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("zone_replication_enabled"),
						knownvalue.NotNull(),
					),
				},
			},
			// update and read
			{
				Config: testAccResourceXelonObjectStorageUserConfig(nameUpdated, 500),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(nameUpdated),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("region"),
						knownvalue.StringExact("zh1"),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("s3_endpoints"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("storage_limit"),
						knownvalue.Int64Exact(500),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("tenant_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_user.test",
						tfjsonpath.New("zone_replication_enabled"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccResourceXelonObjectStorageUserConfig(name string, storageLimit int) string {
	return fmt.Sprintf(`
resource "xelon_object_storage_user" "test" {
  name          = %[1]q
  region        = "zh1"
  storage_limit = %[2]d
  tenant_id     =  data.xelon_tenant.test.id
}

data "xelon_tenant" "test" {}
`, name, storageLimit)
}
