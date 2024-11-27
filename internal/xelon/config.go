package xelon

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Xelon-AG/xelon-sdk-go/xelon"
)

// Config is configuration defined in the provider block.
type Config struct {
	BaseURL         string
	ClientID        string
	Token           string
	ProviderVersion string
}

func (c *Config) Client(ctx context.Context) (*xelon.Client, error) {
	if c.Token == "" {
		return nil, errors.New("token must be set")
	}

	opts := []xelon.ClientOption{xelon.WithUserAgent(c.userAgent())}
	opts = append(opts, xelon.WithBaseURL(c.BaseURL))

	if c.ClientID != "" {
		opts = append(opts, xelon.WithClientID(c.ClientID))
	}

	client := xelon.NewClient(c.Token, opts...)

	tflog.Info(ctx, "Xelon SDK client configured", map[string]interface{}{
		"base_url":  c.BaseURL,
		"client_id": c.ClientID,
	})

	return client, nil
}

func (c *Config) userAgent() string {
	name := "terraform-provider-xelon"
	comment := "https://registry.terraform.io/providers/Xelon-AG/xelon"

	return fmt.Sprintf("%s/%s (+%s)", name, c.ProviderVersion, comment)
}
