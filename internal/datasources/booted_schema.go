package datasources

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func grpcResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"ip": {
				Type:     schema.TypeString,
				Required: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"zone": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func BootedDataSource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"grpc": {
				Type:     schema.TypeSet,
				Elem:     grpcResource(),
				Computed: true,
			},
		},
		Description: `Copy an image (manifest, filesystem layers, signatures) from one location to another.
Uses the system's trust policy to validate images, rejects images not trusted by the policy.
source-image and destination-image are interpreted completely independently; e.g. the destination name does not automatically inherit any parts of the source name.`,
		ReadContext: readBooted,
	}
}
