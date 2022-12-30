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

func TestAccResourceXelonNetwork_basic(t *testing.T) {
	var networkInfo xelon.NetworkInfo
	networkName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccResourceXelonNetworkConfig(networkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkExists("xelon_network.test", &networkInfo),
					resource.TestCheckResourceAttrSet("xelon_network.test", "id"),
					resource.TestCheckResourceAttrSet("xelon_network.test", "netmask"),
					resource.TestCheckResourceAttr("xelon_network.test", "gateway", "10.11.12.1"),
					resource.TestCheckResourceAttr("xelon_network.test", "name", networkName),
				),
			},
		},
	})
}

func TestAccResourceXelonNetwork_changeDNS(t *testing.T) {
	var networkInfo xelon.NetworkInfo
	networkName := fmt.Sprintf("%s-%s", accTestPrefix, acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkDestroy,

		Steps: []resource.TestStep{
			{
				Config: testAccResourceXelonNetworkConfig(networkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkExists("xelon_network.test", &networkInfo),
					resource.TestCheckResourceAttrSet("xelon_network.test", "id"),
					resource.TestCheckResourceAttrSet("xelon_network.test", "netmask"),
					resource.TestCheckResourceAttr("xelon_network.test", "dns_primary", "1.1.1.1"),
					resource.TestCheckResourceAttr("xelon_network.test", "dns_secondary", "2.2.2.2"),
					resource.TestCheckResourceAttr("xelon_network.test", "name", networkName),
				),
			},
			{
				Config: testAccResourceXelonNetworkConfig_changeDNS(networkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkExists("xelon_network.test", &networkInfo),
					resource.TestCheckResourceAttr("xelon_network.test", "dns_primary", "8.8.8.8"),
					resource.TestCheckResourceAttr("xelon_network.test", "dns_secondary", "8.8.8.8"),
				),
			},
		},
	})
}

func testAccCheckNetworkExists(n string, networkInfo *xelon.NetworkInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no network ID is set")
		}

		client := testAccProvider.Meta().(*xelon.Client)
		ctx := context.Background()

		tenant, err := fetchTenant(ctx, client)
		if err != nil {
			return fmt.Errorf("could not fetch tenant: %s", err)
		}
		networkID, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("invalid network id: %v", err)
		}

		retrievedNetworkInfo, _, err := client.Networks.Get(ctx, tenant.TenantID, networkID)
		if err != nil {
			return err
		}

		if retrievedNetworkInfo.Details.NetworkID != networkID {
			return fmt.Errorf("network not found")
		}

		networkInfo = retrievedNetworkInfo
		return nil
	}
}

func testAccCheckNetworkDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*xelon.Client)
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "xelon_network" {
			continue
		}

		tenant, err := fetchTenant(ctx, client)
		if err != nil {
			return fmt.Errorf("could not fetch tenant: %s", err)
		}
		networkID, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("invalid network id: %v", err)
		}

		networkInfo, _, err := client.Networks.Get(ctx, tenant.TenantID, networkID)
		if err == nil && networkInfo.Details.ID == networkID {
			return fmt.Errorf("network (%s) still exists", rs.Primary.ID)
		}
	}

	return nil
}

func testAccResourceXelonNetworkConfig(name string) string {
	return fmt.Sprintf(`
resource "xelon_network" "test" {
  cloud_id      = 4
  dns_primary   = "1.1.1.1"
  dns_secondary = "2.2.2.2"
  gateway       = "10.11.12.1"
  name          = "%s"
  network       = "10.11.12.0"
  type          = "LAN"
}
`, name)
}

func testAccResourceXelonNetworkConfig_changeDNS(name string) string {
	return fmt.Sprintf(`
resource "xelon_network" "test" {
  cloud_id      = 4
  dns_primary   = "8.8.8.8"
  dns_secondary = "8.8.8.8"
  gateway       = "10.11.12.1"
  name          = "%s"
  network       = "10.11.12.0"
  type          = "LAN"
}
`, name)
}
