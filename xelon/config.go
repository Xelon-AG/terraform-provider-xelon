package xelon

import (
	"fmt"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

// Config is configuration defined in the provider block.
type Config struct {
	BaseURL          string
	Token            string
	TerraformVersion string
}

func (c *Config) Client() *xelon.Client {
	client := xelon.NewClient(c.Token)
	client.SetBaseURL(c.BaseURL)

	// TODO: extract using ldflags
	providerVersion := "terraform-provider-xelon/dev"
	userAgent := fmt.Sprintf("Terraform/%s (https://www.terraform.io) %s", c.TerraformVersion, providerVersion)
	client.SetUserAgent(userAgent)

	return client
}
