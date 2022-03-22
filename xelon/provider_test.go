package xelon

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProvider *schema.Provider
var testAccProviders map[string]*schema.Provider
var testAccProviderFactories map[string]func() (*schema.Provider, error)

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"xelon": testAccProvider,
	}
	testAccProviderFactories = map[string]func() (*schema.Provider, error){
		"xelon": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("XELON_BASE_URL"); v == "" {
		t.Fatal("XELON_BASE_URL must be set for acceptance tests")
	}

	if v := os.Getenv("XELON_TOKEN"); v == "" {
		t.Fatal("XELON_TOKEN must be set for acceptance tests")
	}
}
