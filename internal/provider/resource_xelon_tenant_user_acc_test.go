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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func init() {
	resource.AddTestSweepers("xelon_tenant_user", &resource.Sweeper{
		Name: "xelon_tenant_user",
		F: func(region string) error {
			ctx := context.Background()
			client, err := sharedClient(region)
			if err != nil {
				return err
			}

			tenant, _, err := client.Tenants.GetCurrent(ctx)
			if err != nil {
				return fmt.Errorf("getting current tenant: %w", err)
			}

			users, errf := client.TenantUsers.All(ctx, tenant.ID, &xelon.ListOptions{PerPage: 100})
			for user := range users {
				if !user.IsActive || !strings.HasPrefix(user.Email, accTestPrefix) {
					continue
				}

				slog.Info("Deleting xelon_tenant_user", "email", user.Email, "id", user.ID)
				_, err := client.TenantUsers.Delete(ctx, tenant.ID, user.ID)
				if err != nil {
					slog.Warn("Error deleting tenant user during sweep", "email", user.Email, "id", user.ID, "error", err)
				}
			}
			if err := errf(); err != nil {
				return fmt.Errorf("getting tenant user list: %w", err)
			}
			return nil
		},
	})
}

func TestAccResourceXelonTenantUser(t *testing.T) {
	name := acctest.RandomWithPrefix(accTestPrefix)
	email := fmt.Sprintf("%s@xelon.ch", name)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create and read
			{
				Config: testAccResourceXelonTenantUserConfig(email, "Terraform", "User"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_tenant_user.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_tenant_user.test",
						tfjsonpath.New("tenant_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_tenant_user.test",
						tfjsonpath.New("email"),
						knownvalue.StringExact(email),
					),
					statecheck.ExpectKnownValue(
						"xelon_tenant_user.test",
						tfjsonpath.New("first_name"),
						knownvalue.StringExact("Terraform"),
					),
					statecheck.ExpectKnownValue(
						"xelon_tenant_user.test",
						tfjsonpath.New("last_name"),
						knownvalue.StringExact("User"),
					),
					statecheck.ExpectKnownValue(
						"xelon_tenant_user.test",
						tfjsonpath.New("roles"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("hq_root_admin"),
						}),
					),
					statecheck.ExpectKnownValue(
						"xelon_tenant_user.test",
						tfjsonpath.New("permissions"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("allow_view_virtual_machines"),
						}),
					),
				},
			},
			// Import the user and verify API-recoverable state.
			{
				ResourceName: "xelon_tenant_user.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["xelon_tenant_user.test"]
					if !ok {
						return "", fmt.Errorf("not found: xelon_tenant_user.test")
					}
					tenantID := rs.Primary.Attributes["tenant_id"]
					userID := rs.Primary.Attributes["id"]
					return fmt.Sprintf("%v/%v", tenantID, userID), nil
				},
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
					"require_password_change",
					"send_welcome_email",
				},
			},
		},
	})
}

func testAccResourceXelonTenantUserConfig(email, firstName, lastName string) string {
	return fmt.Sprintf(`
resource "xelon_tenant_user" "test" {
  tenant_id  = data.xelon_tenant.test.id
  email      = %[1]q
  first_name = %[2]q
  last_name  = %[3]q
  password   = "x#hahaha6fd_Dx9"

  roles       = ["hq_root_admin"]
  permissions = ["allow_view_virtual_machines"]
}

data "xelon_tenant" "test" {
  id = "c4c59cf3a0fe"
}
`, email, firstName, lastName)
}
