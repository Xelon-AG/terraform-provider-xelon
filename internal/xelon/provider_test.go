package xelon

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

const accTestPrefix = "tf-acc-test"

var testAccProvider = New("testacc")()
var testAccProviderFactories = map[string]func() (*schema.Provider, error){
	"xelon": func() (*schema.Provider, error) {
		return testAccProvider, nil
	},
}

func TestProvider(t *testing.T) {
	err := testAccProvider.InternalValidate()

	assert.NoError(t, err)
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
