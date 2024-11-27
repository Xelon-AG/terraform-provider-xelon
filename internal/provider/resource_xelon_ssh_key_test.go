package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
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

	sshKeys, _, err := client.SSHKeys.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("getting SSH key list: %s", err)
	}

	for _, sshKey := range sshKeys {
		if strings.HasPrefix(sshKey.Name, accTestPrefix) {
			slog.Info("Deleting xelon_ssh_key", "name", sshKey.Name, "id", sshKey.ID)
			_, err := client.SSHKeys.Delete(ctx, sshKey.ID)
			if err != nil {
				slog.Warn("Error deleting SSH key during sweep", "name", sshKey.Name, "error", err)
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

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSSHKeyDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccXelonSSHKeyResource(sshKeyName, sshKeyPublic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSSHKeyExists("xelon_ssh_key.foobar", &sshKey),
					resource.TestCheckResourceAttr("xelon_ssh_key.foobar", "name", sshKeyName),
					resource.TestCheckResourceAttrSet("xelon_ssh_key.foobar", "id"),
					resource.TestCheckResourceAttrSet("xelon_ssh_key.foobar", "public_key"),
				),
			},
		},
	})
}

func TestAccResourceXelonSSHKey_update(t *testing.T) {
	var sshKey xelon.SSHKey
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyNameUpdated := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyPublic, _, err := acctest.RandSSHKeyPair("xelon@ssh-acceptance-test")
	if err != nil {
		t.Fatalf("could not generate test SSH key: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSSHKeyDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccXelonSSHKeyResource(sshKeyName, sshKeyPublic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSSHKeyExists("xelon_ssh_key.foobar", &sshKey),
					resource.TestCheckResourceAttr("xelon_ssh_key.foobar", "name", sshKeyName),
					resource.TestCheckResourceAttrSet("xelon_ssh_key.foobar", "id"),
					resource.TestCheckResourceAttrSet("xelon_ssh_key.foobar", "public_key"),
				),
			},
			{
				Config: testAccXelonSSHKeyResource(sshKeyNameUpdated, sshKeyPublic),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSSHKeyExists("xelon_ssh_key.foobar", &sshKey),
					resource.TestCheckResourceAttr("xelon_ssh_key.foobar", "name", sshKeyNameUpdated),
					resource.TestCheckResourceAttrSet("xelon_ssh_key.foobar", "id"),
					resource.TestCheckResourceAttrSet("xelon_ssh_key.foobar", "public_key"),
				),
			},
		},
	})
}

func TestAccResourceXelonSSHKey_expectError(t *testing.T) {
	sshKeyName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))
	sshKeyPublic, _, err := acctest.RandSSHKeyPair("xelon@ssh-acceptance-test")
	if err != nil {
		t.Fatalf("could not generate test SSH key: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckSSHKeyDestroy,

		Steps: []resource.TestStep{
			{
				Config:      testAccXelonSSHKeyResourceWithoutName(sshKeyPublic),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
			{
				Config:      testAccXelonSSHKeyResourceWithoutPublicKey(sshKeyName),
				ExpectError: regexp.MustCompile(`The argument "public_key" is required`),
			},
		},
	})
}

func testAccCheckSSHKeyDestroy(s *terraform.State) error {
	ctx := context.Background()
	client, err := sharedClient("testacc")
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "xelon_ssh_key" {
			continue
		}

		sshKey, _, err := client.SSHKeys.Get(ctx, rs.Primary.ID)
		if err == nil && sshKey.ID == rs.Primary.ID {
			return fmt.Errorf("SSH key (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccCheckSSHKeyExists(n string, sshKey *xelon.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("no SSH key ID set")
		}

		ctx := context.Background()
		client, err := sharedClient("testacc")
		if err != nil {
			return err
		}

		retrievedSSHKey, _, err := client.SSHKeys.Get(ctx, rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("could not get SSH key: %s", err)
		}

		if retrievedSSHKey.ID != rs.Primary.ID {
			return fmt.Errorf("could not found SSH key with ID: %s", rs.Primary.ID)
		}

		sshKey = retrievedSSHKey
		return nil
	}
}

func testAccXelonSSHKeyResource(name, publicKey string) string {
	return fmt.Sprintf(`
resource "xelon_ssh_key" "foobar" {
  name       = "%s"
  public_key = "%s"
}`, name, publicKey)
}

func testAccXelonSSHKeyResourceWithoutName(publicKey string) string {
	return fmt.Sprintf(`
resource "xelon_ssh_key" "foobar" {
  public_key = "%s"
}`, publicKey)
}

func testAccXelonSSHKeyResourceWithoutPublicKey(name string) string {
	return fmt.Sprintf(`
resource "xelon_ssh_key" "foobar" {
  name = "%s"
}`, name)
}
