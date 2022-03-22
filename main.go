package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"github.com/Xelon-AG/terraform-provider-xelon/xelon"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: xelon.Provider,
	})
}
