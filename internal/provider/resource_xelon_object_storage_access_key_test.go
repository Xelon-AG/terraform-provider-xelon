package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccResourceXelonObjectStorageAccessKey(t *testing.T) {
	name := acctest.RandomWithPrefix(accTestPrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read
			{
				Config: testAccResourceXelonObjectStorageAccessKeyConfig(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_object_storage_access_key.test",
						tfjsonpath.New("access_key_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_access_key.test",
						tfjsonpath.New("created_at"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_access_key.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_access_key.test",
						tfjsonpath.New("secret_access_key"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_object_storage_access_key.test",
						tfjsonpath.New("user_id"),
						knownvalue.NotNull(),
					),
				},
			},
			// ImportState testing
			{
				ResourceName: "xelon_object_storage_access_key.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["xelon_object_storage_access_key.test"]
					if !ok {
						return "", fmt.Errorf("not found: xelon_object_storage_access_key.test")
					}
					userID := rs.Primary.Attributes["user_id"]
					tokenID := rs.Primary.Attributes["id"]
					return fmt.Sprintf("%v/%v", userID, tokenID), nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret_access_key"},
			},
		},
	})
}

func testAccResourceXelonObjectStorageAccessKeyConfig(name string) string {
	return fmt.Sprintf(`
resource "xelon_object_storage_access_key" "test" {
  user_id = xelon_object_storage_user.test.id
}

resource "xelon_object_storage_user" "test" {
  name          = %[1]q
  region        = "zh1"
  storage_limit = 100
  tenant_id     =  data.xelon_tenant.test.id
}

data "xelon_tenant" "test" {}
`, name)
}
