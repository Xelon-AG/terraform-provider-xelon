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

func TestAccDataSourceXelonSSHKey(t *testing.T) {
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyPublic, _, err := acctest.RandSSHKeyPair(sshKeyName)
	if err != nil {
		t.Errorf("failed to generate SSH key for acceptance test: %v", err.Error())
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceXelonSSHKeyConfig(sshKeyName, sshKeyPublic),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.xelon_ssh_key.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_ssh_key.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(sshKeyName),
					),
					statecheck.ExpectKnownValue(
						"data.xelon_ssh_key.test",
						tfjsonpath.New("public_key"),
						knownvalue.StringExact(sshKeyPublic),
					),
				},
			},
		},
	})
}

func TestAccDataSourceXelonSSHKey_expectError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceXelonSSHKeyConfigWithoutIDAndName,
				ExpectError: regexp.MustCompile(`The attribute "id" or "name" must be defined`),
			},
		},
	})
}

func testAccDataSourceXelonSSHKeyConfig(name, publicKey string) string {
	return fmt.Sprintf(`
data "xelon_ssh_key" "test" {
  id = xelon_ssh_key.test.id
}

resource "xelon_ssh_key" "test" {
  name       = %[1]q
  public_key = %[2]q
}
`, name, publicKey)
}

const testAccDataSourceXelonSSHKeyConfigWithoutIDAndName = `
data "xelon_ssh_key" "test" {}
`
