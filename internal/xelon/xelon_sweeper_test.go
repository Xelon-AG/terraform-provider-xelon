package xelon

// import (
// 	"testing"
//
// 	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
// )

// func TestMain(m *testing.M) {
// 	resource.TestMain(m)
// }

// func sharedClient(_ string) (*xelon.Client, error) {
// 	if v := os.Getenv("XELON_BASE_URL"); v == "" {
// 		return nil, fmt.Errorf("empty XELON_BASE_URL")
// 	}
// 	if v := os.Getenv("XELON_CLIENT_ID"); v == "" {
// 		return nil, fmt.Errorf("empty XELON_CLIENT_ID")
// 	}
// 	if v := os.Getenv("XELON_TOKEN"); v == "" {
// 		return nil, fmt.Errorf("empty XELON_TOKEN")
// 	}
//
// 	config := &Config{
// 		BaseURL:         os.Getenv("XELON_BASE_URL"),
// 		ClientID:        os.Getenv("XELON_CLIENT_ID"),
// 		Token:           os.Getenv("XELON_TOKEN"),
// 		ProviderVersion: "sweeper",
// 	}
//
// 	return config.Client(context.Background())
// }
