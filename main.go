package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/xelon"
)

var (
	version = "dev"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: xelon.New(version),
	})
}
