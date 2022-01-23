package datasources

import (
	"context"

	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func readBooted(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	addr := m.(meta.Data).GRPCAddr
	err := d.Set("grpc", []interface{}{
		map[string]interface{}{
			"ip":   addr.IP.String(),
			"port": addr.Port,
			"zone": addr.Zone,
		},
	})
	return diag.FromErr(err)
}
