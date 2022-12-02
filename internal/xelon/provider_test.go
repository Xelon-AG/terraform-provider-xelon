package xelon

//
// import (
// 	"context"
// 	"os"
// 	"testing"
//
// 	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
// 	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
// )
//
// const accTestPrefix = "tf-acc-test-"
//
// var testAccProvider = New("testacc")
// var testAccProviderFactories map[string]func() (*schema.Provider, error)
//
// func init() {
// 	testAccProvider = New("testacc")
// 	testAccProviderFactories = map[string]func() (*schema.Provider, error){
// 		"xelon": func() (*schema.Provider, error) {
// 			return testAccProvider, nil
// 		},
// 	}
// }
//
// func TestProvider(t *testing.T) {
// 	if err := Provider().InternalValidate(); err != nil {
// 		t.Fatalf("err: %s", err)
// 	}
// }
//
// func TestProvider_impl(t *testing.T) {
// 	var _ = Provider()
// }
//
// func TestProvider_BaseURLOverride(t *testing.T) {
// 	customBaseURL := "https://mock-api.internal.xelon.ch/service/"
//
// 	rawProvider := Provider()
// 	raw := map[string]interface{}{
// 		"base_url": customBaseURL,
// 		"token":    "abcdef12345",
// 	}
//
// 	diagnostics := rawProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(raw))
// 	if diagnostics.HasError() {
// 		t.Fatalf("provider configure failed: %v", diagnostics)
// 	}
// }
//
// func testAccPreCheck(t *testing.T) {
// 	if v := os.Getenv("XELON_BASE_URL"); v == "" {
// 		t.Fatal("XELON_BASE_URL must be set for acceptance tests")
// 	}
//
// 	if v := os.Getenv("XELON_TOKEN"); v == "" {
// 		t.Fatal("XELON_TOKEN must be set for acceptance tests")
// 	}
// }
