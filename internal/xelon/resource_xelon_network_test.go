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
	resource.AddTestSweepers("xelon_network", &resource.Sweeper{
		Name: "xelon_network",
		F:    testSweepNetworks,
	})
}

func testSweepNetworks(region string) error {
	ctx := context.Background()
	client, err := sharedClient(region)
	if err != nil {
		return err
	}

	tenant, _, err := client.Tenants.GetCurrent(ctx)
	if err != nil {
		return fmt.Errorf("getting tenant: %s", err)
	}

	networks, _, err := client.Networks.List(ctx, tenant.TenantID)
	if err != nil {
		return fmt.Errorf("getting networks list: %s", err)
	}

	for _, network := range networks {
		if strings.HasPrefix(network.Name, accTestPrefix) {
			log.Printf("[DEBUG] Deleting xelon_network: %s (%d)", network.Name, network.ID)
			_, err := client.Networks.Delete(ctx, network.ID)
			if err != nil {
				log.Printf("Error destroying %s during sweep: %s", network.Name, err)
			}
		}
	}

	return nil
}

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
					resource.TestCheckResourceAttr("xelon_network.test", "cloud_id", "8"),
					resource.TestCheckResourceAttr("xelon_network.test", "gateway", "10.11.12.1"),
					resource.TestCheckResourceAttr("xelon_network.test", "name", networkName),
					resource.TestCheckResourceAttrSet("xelon_network.test", "id"),
					resource.TestCheckResourceAttrSet("xelon_network.test", "netmask"),
				),
			},
		},
	})
}

func TestAccResourceXelonNetwork_update(t *testing.T) {
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
					resource.TestCheckResourceAttr("xelon_network.test", "dns_primary", "1.1.1.1"),
					resource.TestCheckResourceAttr("xelon_network.test", "dns_secondary", "2.2.2.2"),
					resource.TestCheckResourceAttr("xelon_network.test", "name", networkName),
					resource.TestCheckResourceAttrSet("xelon_network.test", "id"),
					resource.TestCheckResourceAttrSet("xelon_network.test", "netmask"),
				),
			},
			{
				Config: testAccResourceXelonNetworkConfigChangeDNS(networkName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkExists("xelon_network.test", &networkInfo),
					resource.TestCheckResourceAttr("xelon_network.test", "dns_primary", "8.8.8.8"),
					resource.TestCheckResourceAttr("xelon_network.test", "dns_secondary", "8.8.8.8"),
					resource.TestCheckResourceAttr("xelon_network.test", "name", networkName),
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

		if retrievedNetworkInfo.Details.ID != networkID {
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
  cloud_id      = 8
  dns_primary   = "1.1.1.1"
  dns_secondary = "2.2.2.2"
  gateway       = "10.11.12.1"
  name          = "%s"
  network       = "10.11.12.0"
  type          = "LAN"
}
`, name)
}

func testAccResourceXelonNetworkConfigChangeDNS(name string) string {
	return fmt.Sprintf(`
resource "xelon_network" "test" {
  cloud_id      = 8
  dns_primary   = "8.8.8.8"
  dns_secondary = "8.8.8.8"
  gateway       = "10.11.12.1"
  name          = "%s"
  network       = "10.11.12.0"
  type          = "LAN"
}
`, name)
}
