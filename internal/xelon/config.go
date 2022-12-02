package xelon

import (
	"fmt"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

// Config is configuration defined in the provider block.
type Config struct {
	BaseURL         string
	Token           string
	ProviderVersion string
}

func (c *Config) Client() *xelon.Client {
	opts := []xelon.ClientOption{xelon.WithUserAgent(c.userAgent())}
	opts = append(opts, xelon.WithBaseURL(c.BaseURL))

	client := xelon.NewClient(c.Token, opts...)

	return client
}

func (c *Config) userAgent() string {
	name := "terraform-provider-xelon"
	comment := "https://registry.terraform.io/providers/xelon-ag/xelon"

	return fmt.Sprintf("%s/%s (+%s)", name, c.ProviderVersion, comment)
}
