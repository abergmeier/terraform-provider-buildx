package datasources

import (
	"context"

	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type nodeGroupDataSource struct {
	m *meta.Data
}

type nodeGroupData struct {
	Name      types.String `tfsdk:"name"`
	Endpoints types.Set    `tfsdk:"endpoints"`
}

func (ds *nodeGroupDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var d nodeGroupData

	diags := req.Config.Get(ctx, &d)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	md, err := readNodeGroupByName(ctx, nil, d.Name.Value)
	if err != nil {
		resp.Diagnostics.AddError("readNodeGroupByName failed", err.Error())
		return
	}

	elems := []attr.Value{}
	for k := range md.Endpoints {
		elems = append(elems, types.String{
			Value: k,
		})
	}
	d.Endpoints.Elems = elems
	diags = resp.State.Set(ctx, &d)
	resp.Diagnostics.Append(diags...)
}
