package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/Masumoou/terraform-provider-nextcloud/internal/provider"
)

// Version is set at build time via -ldflags "-X main.version=x.y.z"
var version string = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		// Registry address once/if this is published internally or publicly.
		Address: "registry.terraform.io/Masumoou/nextcloud",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
