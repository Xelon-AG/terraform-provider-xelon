package xelon

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func init() {
	resource.AddTestSweepers("xelon_ssh_key", &resource.Sweeper{
		Name: "xelon_ssh_key",
		F:    testSweepSSHKeys,
	})
}

func testSweepSSHKeys(region string) error {
	ctx := context.Background()
	client, err := sharedClient(region)
	if err != nil {
		return err
	}

	sshKeys, _, err := client.SSHKeys.List(ctx)
	if err != nil {
		return fmt.Errorf("getting ssh keys list: %s", err)
	}

	for _, sshKey := range sshKeys {
		if strings.HasPrefix(sshKey.Name, accTestPrefix) {
			log.Printf("[DEBUG] Deleting xelon_ssh_key: %s (%d)", sshKey.Name, sshKey.ID)
			_, err := client.SSHKeys.Delete(ctx, sshKey.ID)
			if err != nil {
				log.Printf("Error destroying %s during sweep: %s", sshKey.Name, err)
			}
		}
	}

	return nil
}

func TestAccResourceXelonSSHKey_basic(t *testing.T) {
	var sshKey xelon.SSHKey
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyPublic, _, err := acctest.RandSSHKeyPair("xelon@ssh-acceptance-test")
	if err != nil {
		t.Fatalf("could not generate test SSH key: %s", err)
	}

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
			retrievedSSHKey := retrievedSSHKey
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
