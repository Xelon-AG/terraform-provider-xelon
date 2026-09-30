package provider

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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

func TestAccResourceXelonSSHKey(t *testing.T) {
	sshKeyName := acctest.RandomWithPrefix(accTestPrefix)
	sshKeyNameUpdated := acctest.RandomWithPrefix(accTestPrefix)
	sshKeyPublic, _, err := acctest.RandSSHKeyPair("xelon@ssh-acceptance-test")
	if err != nil {
		t.Fatalf("could not generate test SSH key: %s", err)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccXelonSSHKeyResourceConfig(sshKeyName, sshKeyPublic),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_ssh_key.foobar",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_ssh_key.foobar",
						tfjsonpath.New("name"),
						knownvalue.StringExact(sshKeyName),
					),
					statecheck.ExpectKnownValue(
						"xelon_ssh_key.foobar",
						tfjsonpath.New("public_key"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				ResourceName:      "xelon_ssh_key.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccXelonSSHKeyResourceConfig(sshKeyNameUpdated, fmt.Sprintf("%s\\n\\n", sshKeyPublic)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"xelon_ssh_key.foobar",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"xelon_ssh_key.foobar",
						tfjsonpath.New("name"),
						knownvalue.StringExact(sshKeyNameUpdated),
					),
					statecheck.ExpectKnownValue(
						"xelon_ssh_key.foobar",
						tfjsonpath.New("public_key"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccResourceXelonSSHKey_expectError(t *testing.T) {
	sshKeyName := acctest.RandomWithPrefix(accTestPrefix)
	sshKeyPublic, _, err := acctest.RandSSHKeyPair("xelon@ssh-acceptance-test")
	if err != nil {
		t.Fatalf("could not generate test SSH key: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

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

func testAccXelonSSHKeyResourceConfig(name, publicKey string) string {
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
