package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func sharedClient(_ string) (*xelon.Client, error) {
	if v := os.Getenv("XELON_BASE_URL"); v == "" {
		return nil, fmt.Errorf("empty XELON_BASE_URL")
	}
	if v := os.Getenv("XELON_CLIENT_ID"); v == "" {
		return nil, fmt.Errorf("empty XELON_CLIENT_ID")
	}
	if v := os.Getenv("XELON_TOKEN"); v == "" {
		return nil, fmt.Errorf("empty XELON_TOKEN")
	}

	baseURL := os.Getenv("XELON_BASE_URL")
	clientID := os.Getenv("XELON_CLIENT_ID")
	token := os.Getenv("XELON_TOKEN")

	opts := []xelon.ClientOption{
		xelon.WithUserAgent("terraform-provider-xelon/sweeper"),
		xelon.WithBaseURL(baseURL),
		xelon.WithClientID(clientID),
	}

	return xelon.NewClient(token, opts...), nil
}
