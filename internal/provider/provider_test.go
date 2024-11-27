package provider

import (
	"context"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/xelon"
)

const accTestPrefix = "tf-acc-test"

var testAccProvider = New("testacc")()
var testAccProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"xelon": func() (tfprotov6.ProviderServer, error) {
		ctx := context.Background()

		upgradedSDKProvider, err := tf5to6server.UpgradeServer(ctx, xelon.New("testacc")().GRPCProvider)
		if err != nil {
			return nil, err
		}
		providers := []func() tfprotov6.ProviderServer{
			providerserver.NewProtocol6(testAccProvider),
			func() tfprotov6.ProviderServer { return upgradedSDKProvider },
		}
		muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
		if err != nil {
			return nil, err
		}

		return muxServer.ProviderServer(), nil
	},
}

// var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
// 	"xelon": providerserver.NewProtocol6WithError(New("testacc")()),
// }

func TestProvider_MissingTokenAttribute(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviderFactories,

		Steps: []resource.TestStep{
			{
				Config:      testProviderConfigWithMissingToken,
				ExpectError: regexp.MustCompile(`token must be set`),
			},
		},
	})
}

func TestProvider_userAgent(t *testing.T) {
	t.Parallel()

	type testCase struct {
		version           string
		expectedUserAgent string
	}
	tests := map[string]testCase{
		"empty_version": {
			version:           "",
			expectedUserAgent: "terraform-provider-xelon/ (+https://registry.terraform.io/providers/Xelon-AG/xelon)",
		},
		"dev_version": {
			version:           "dev",
			expectedUserAgent: "terraform-provider-xelon/dev (+https://registry.terraform.io/providers/Xelon-AG/xelon)",
		},
		"release_version": {
			version:           "1.1.1",
			expectedUserAgent: "terraform-provider-xelon/1.1.1 (+https://registry.terraform.io/providers/Xelon-AG/xelon)",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := &xelonProvider{version: test.version}
			actualUserAgent := p.userAgent()

			assert.Equal(t, test.expectedUserAgent, actualUserAgent)
		})
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("XELON_BASE_URL"); v == "" {
		t.Fatal("XELON_BASE_URL must be set for acceptance tests")
	}

	if v := os.Getenv("XELON_CLIENT_ID"); v == "" {
		t.Fatal("XELON_CLIENT_ID must be set for acceptance tests")
	}

	if v := os.Getenv("XELON_TOKEN"); v == "" {
		t.Fatal("XELON_TOKEN must be set for acceptance tests")
	}
}

const testProviderConfigWithMissingToken = `
provider "xelon" {
  token = ""
}
data "xelon_network" "test" {
  filter = {
    network_id = 1
  }
}
`
