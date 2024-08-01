package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceNetwork_basic(t *testing.T) {
	t.Skip()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNetworkConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.xelon_network.test", "id", "93"),
					resource.TestCheckResourceAttr("data.xelon_network.test", "network_id", "93"),
					resource.TestCheckResourceAttr("data.xelon_network.test", "network", "10.0.0.0"),
				),
			},
		},
	})
}

func TestAccDataSourceNetwork_missingNetworkID(t *testing.T) {
	t.Skip()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceNetworkWithMissingNetworkID,
				ExpectError: regexp.MustCompile(`Attribute filter\.network_id must be set`),
			},
		},
	})
}

const testAccDataSourceNetworkConfig = `
data "xelon_network" "test" {
  filter = {
    # 93 is the network from test environment
    network_id = 93
  }
}
`

const testAccDataSourceNetworkWithMissingNetworkID = `
data "xelon_network" "test" {
  filter = {
    # network_id is mandatory
  }
}
`
