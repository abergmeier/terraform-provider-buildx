package testproviderfactory

import (
	"github.com/abergmeier/terraform-provider-buildx/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var (
	SingleFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"buildx": func() (tfprotov6.ProviderServer, error) {
			// newProvider is your function that returns a
			// tfsdk.Provider implementation
			return providerserver.NewProtocol6(provider.New("1.0")())(), nil
		},
	}
)
