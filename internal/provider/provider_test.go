package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

const accTestPrefix = "tf-acc-test"

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"xelon": providerserver.NewProtocol6WithError(New("testacc")()),
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

func TestProvider_MissingTokenAttribute(t *testing.T) {
	t.Skip("refactoring to framework")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config:      testProviderConfigWithMissingToken,
				ExpectError: regexp.MustCompile(`token must be set`),
			},
		},
	})
}

func TestProvider_userAgent(t *testing.T) {
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
