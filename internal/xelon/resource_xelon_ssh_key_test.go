package xelon

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceXelonSSHKey_basic(t *testing.T) {
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyPublic := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAA/yupp+bxv9EKJmg5LNwu1foNjby/Nl++Nx2XTmi80BRRa4daLNQYJ7oQ=="
	config := fmt.Sprintf(testAccResourceXelonSSHKeyConfig, sshKeyName, sshKeyPublic)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("xelon_ssh_key.default", "id"),
					resource.TestCheckResourceAttr("xelon_ssh_key.default", "name", sshKeyName),
				),
			},
		},
	})
}

const testAccResourceXelonSSHKeyConfig = `
resource "xelon_ssh_key" "default" {
  name = "%s"
  public_key = "%s"
}
`
