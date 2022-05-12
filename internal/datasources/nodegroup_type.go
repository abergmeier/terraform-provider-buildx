package datasources

import (
	"context"

	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	NodeGroupType tfsdk.DataSourceType = &nodeGroupType{}
)

type nodeGroupType struct{}

func (t *nodeGroupType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Type:     types.StringType,
				Required: true,
			},
			"endpoints": {
				Type: types.SetType{
					ElemType: types.StringType,
				},
				Computed: true,
			},
		},
	}, nil
}

func (t *nodeGroupType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	return &nodeGroupDataSource{
		m: p.(meta.HasData).Data(),
	}, nil
}
