package resources

import (
	"context"

	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type builtResource struct {
	m *meta.Data
}

func (r *builtResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	dockerCli := r.m.Cli

	data := builtResourceData{}
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var cacheFrom, cacheTo []cacheEntryData
	if data.Cache != nil {
		cacheFrom = data.Cache.From
		cacheTo = data.Cache.To
	}

	outputs, err := toOutputOptions(data.Output)
	if err != nil {
		ap := tftypes.NewAttributePath().WithAttributeName("output")
		resp.Diagnostics.AddAttributeError(ap, "toOutputOptions failed", err.Error())
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Instance.Null {
		panic("Null builder instance")
	}

	if data.Context.Null {
		panic("Null context")
	}

	if data.File.Null {
		panic("Null file")
	}

	allow := []string{}
	diags = data.Allow.ElementsAs(ctx, &allow, false)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	opt := buildOptions{
		commonOptions: commonOptions{
			builder: data.Instance.Value,
		},
		allow:          allow,
		cacheFrom:      toCacheEntry(cacheFrom),
		cacheTo:        toCacheEntry(cacheTo),
		contextPath:    data.Context.Value,
		dockerfileName: data.File.Value,
		buildArgs:      data.BuildArgs,
		labels:         data.Labels,
		tags:           data.Tags,
		outputs:        outputs,
	}

	res, err := createBuiltWithOptions(dockerCli, opt)
	if err != nil {
		resp.Diagnostics.AddError("createBuiltWithOptions failed", err.Error())
		return
	}

	data.Iid = types.String{
		Value: res.imageID,
	}
	data.Metadata = types.String{
		Value: res.metadata,
	}

	// No better idea than to generate an id
	uuid, err := uuid.NewRandom()
	if err != nil {
		resp.Diagnostics.AddError("Generating uuid failed", err.Error())
		return
	}
	data.Id = types.String{
		Value: uuid.String(),
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r *builtResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {

	data := &builtResourceData{}
	diags := req.State.Get(ctx, data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	outputs, err := toOutputOptions(data.Output)
	if err != nil {
		ap := tftypes.NewAttributePath().WithAttributeName("output")
		resp.Diagnostics.AddAttributeError(ap, "toOutputOptions failed", err.Error())
		return
	}

	ctx = tflog.With(ctx, "iid", data.Iid)
	for _, output := range outputs {
		if data.Iid.Null {
			panic("Null iid")
		}
		err := deleteBuiltImage(ctx, r.m.Cli, output, data.Iid.Value)
		if err != nil {
			resp.Diagnostics.AddError("deleteBuiltImage failed", err.Error())
			return
		}
	}
}

func (r *builtResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// TODO: Read local state
}

func (r *builtResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {

}

func (r *builtResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	panic("Internal error - Update should never be called")
}
