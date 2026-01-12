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
	resource.AddTestSweepers("xelon_persistent_storage", &resource.Sweeper{
		Name: "xelon_persistent_storage",
		F:    testSweepPersistentStorages,
	})
}

func testSweepPersistentStorages(region string) error {
	ctx := context.Background()
	client, err := sharedClient(region)
	if err != nil {
		return err
	}

	storages, _, err := client.PersistentStorages.List(ctx, &xelon.PersistentStorageListOptions{ListOptions: xelon.ListOptions{PerPage: 100}})
	if err != nil {
		return fmt.Errorf("getting persistent storage list: %s", err)
	}

	for _, storage := range storages {
		if strings.HasPrefix(storage.Name, accTestPrefix) {
			slog.Info("Deleting xelon_persistent_storage", "name", storage.Name, "id", storage.ID)
			_, err := client.PersistentStorages.Delete(ctx, storage.ID)
			if err != nil {
				slog.Warn("Error deleting persistent storage during sweep", "name", storage.Name, "error", err)
			}
		}
	}

	return nil
}

func TestAccResourceXelonPersistentStorage(t *testing.T) {
	storageName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read
			{
				Config: testAccResourceXelonPersistentStorageConfig(storageName, 10),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("cloud_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("device_id"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(storageName),
					),
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("size"),
						knownvalue.Int64Exact(10),
					),
				},
			},
			// update and read
			{
				Config: testAccResourceXelonPersistentStorageConfig(storageName, 15),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("cloud_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("device_id"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(storageName),
					),
					statecheck.ExpectKnownValue(
						"xelon_persistent_storage.test",
						tfjsonpath.New("size"),
						knownvalue.Int64Exact(15),
					),
				},
			},
		},
	})
}

func testAccResourceXelonPersistentStorageConfig(name string, size int) string {
	return fmt.Sprintf(`
resource "xelon_persistent_storage" "test" {
  cloud_id = data.xelon_cloud.test.id
  name     = %[1]q
  size     = %[2]d
}

data "xelon_cloud" "test" {
  # cloud id from test environment, change if updated
  id = "c9ab2f0fcfde"
}
`, name, size)
}
