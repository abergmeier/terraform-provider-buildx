package instancetest

import (
	"github.com/abergmeier/terraform-provider-buildx/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	testAccProvider  = provider.Provider()
	testAccProviders = map[string]*schema.Provider{
		"buildx": testAccProvider,
	}
)
