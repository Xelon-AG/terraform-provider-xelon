package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6/tf6server"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"

	"github.com/Xelon-AG/terraform-provider-xelon/internal/provider"
	"github.com/Xelon-AG/terraform-provider-xelon/internal/xelon"
)

var (
	version = "dev"
)

func main() {
	var debugMode bool
	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with debug support")
	flag.Parse()

	ctx := context.Background()

	upgradedSDKProvider, err := tf5to6server.UpgradeServer(ctx, xelon.New(version)().GRPCProvider)
	if err != nil {
		log.Fatal(err)
	}
	providers := []func() tfprotov6.ProviderServer{
		func() tfprotov6.ProviderServer {
			return upgradedSDKProvider
		},
		providerserver.NewProtocol6(provider.New(version)()),
	}

	muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
	if err != nil {
		log.Fatal(err)
	}

	var serveOpts []tf6server.ServeOpt
	if debugMode {
		serveOpts = append(serveOpts, tf6server.WithManagedDebug())
	}

	err = tf6server.Serve(
		"registry.terraform.io/Xelon-AG/xelon",
		muxServer.ProviderServer,
		serveOpts...,
	)
	if err != nil {
		log.Fatal(err)
	}
}
