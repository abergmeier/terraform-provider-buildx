package main

import (
	"context"
	"flag"
	"log"

	"github.com/abergmeier/terraform-provider-buildx/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/abergmeier/buildx",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New("0.0.1"), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
