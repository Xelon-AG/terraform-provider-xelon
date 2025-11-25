package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider"
)

var (
	version = "dev"
)

func main() {
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with debug support")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/Xelon-AG/xelon",
		Debug:   debugMode,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
