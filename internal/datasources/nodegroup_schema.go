package datasources

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func NodeGroupDataSource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"endpoints": {
				Type:     schema.TypeSet,
				Elem:     schema.TypeString,
				Computed: true,
			},
		},
		ReadContext: readNodeGroup,
	}
}
