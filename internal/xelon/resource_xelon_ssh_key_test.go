package xelon

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceXelonSSHKey_basic(t *testing.T) {
	var sshKey xelon.SSHKey
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyPublic := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAA/yupp+bxv9EKJmg5LNwu1foNjby/Nl++Nx2XTmi80BRRa4daLNQYJ7oQ=="

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSSHKeyDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccResourceXelonSSHKeyConfig(sshKeyName, sshKeyPublic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSSHKeyExists("xelon_ssh_key.test", &sshKey),
					resource.TestCheckResourceAttrSet("xelon_ssh_key.test", "id"),
					resource.TestCheckResourceAttrSet("xelon_ssh_key.test", "fingerprint"),
					resource.TestCheckResourceAttr("xelon_ssh_key.test", "name", sshKeyName),
					resource.TestCheckResourceAttr("xelon_ssh_key.test", "public_key", sshKeyPublic),
				),
			},
		},
	})
}

func testAccCheckSSHKeyExists(n string, sshKey *xelon.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no SSH Key ID is set")
		}

		client := testAccProvider.Meta().(*xelon.Client)
		retrievedSSHKeys, _, err := client.SSHKeys.List(context.Background())
		if err != nil {
			return err
		}

		for _, retrievedSSHKey := range retrievedSSHKeys {
			sshKeyID := strconv.Itoa(retrievedSSHKey.ID)
			if sshKeyID == rs.Primary.ID {
				sshKey = &retrievedSSHKey
				return nil
			}
		}

		return fmt.Errorf("SSH Key not found")
	}
}

func testAccCheckSSHKeyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*xelon.Client)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "xelon_ssh_key" {
			continue
		}

		sshKeys, _, err := client.SSHKeys.List(context.Background())
		if err != nil {
			return err
		}

		for _, sshKey := range sshKeys {
			sshKeyID := strconv.Itoa(sshKey.ID)
			if sshKeyID == rs.Primary.ID {
				return fmt.Errorf("SSH Key (%s) still exists", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccResourceXelonSSHKeyConfig(name, publicKey string) string {
	return fmt.Sprintf(`
resource "xelon_ssh_key" "test" {
  name = "%s"
  public_key = "%s"
}
`, name, publicKey)
}
