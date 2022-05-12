package resources

import (
	"context"
	"fmt"

	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/docker/buildx/store"
	"github.com/docker/buildx/store/storeutil"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type instanceResource struct {
	m *meta.Data
}

func (r *instanceResource) ValidateConfig(ctx context.Context, req tfsdk.ValidateResourceConfigRequest, resp *tfsdk.ValidateResourceConfigResponse) {

	name := types.String{}
	diags := req.Config.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("name"), &name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Getting Attribute `name` failed")
		return
	}

	generateName := types.Bool{}
	diags = req.Config.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("generate_name"), &generateName)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Getting Attribute `generate_name` failed")
		return
	}

	if generateName.Null {
		generateName = types.Bool{
			Value: false,
		}
	}

	diags = exactlyOneOf("name", func() bool {
		return name.Value != ""
	}, "generate_name", func() bool {
		return *&generateName.Value
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	context := types.String{}
	diags = req.Config.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("context"), &context)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Getting Attribute `context` failed")
		return
	}

	if context.Null {
		context = types.String{
			Value: "",
		}
	}

	endpoint := types.String{}
	diags = req.Config.GetAttribute(ctx, tftypes.NewAttributePath().WithAttributeName("endpoint"), &endpoint)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Getting Attribute `endpoint` failed")
		return
	}

	if endpoint.Null {
		endpoint = types.String{
			Value: "",
		}
	}

	diags = conflictsWithString("context", *&context.Value, "endpoint", *&endpoint.Value)
	resp.Diagnostics.Append(diags...)
}

func (r *instanceResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {

	var d instanceResourceData
	diags := req.Plan.Get(ctx, &d)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Reading Create Attributes failed", map[string]interface{}{
			"plan": req.Plan.Raw,
		})
		return
	}

	dockerCli := r.m.Cli
	txn, release := r.getStore(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	defer release()

	name, err := handleNameAttributes(&d, txn)
	if err != nil {
		resp.Diagnostics.AddError("Handling name attributes failed", err.Error())
		return
	}

	args := []string{}
	if !d.Context.Null && d.Context.Value != "" {
		args = append(args, d.Context.Value)
	} else if !d.Endpoint.Null && d.Endpoint.Value != "" {
		args = append(args, d.Endpoint.Value)
	}

	var flags []string

	if d.Buildkit != nil {
		flags = d.Buildkit.Flags
	}

	bootstrap := false
	if !d.Bootstrap.Null && d.Bootstrap.Value {
		bootstrap = true
	}

	drv := instanceDriverData{}
	if d.Driver != nil {
		drv = *d.Driver
	}

	err = createInstanceFromOptions(ctx, dockerCli, txn, createOptions{
		name:         name,
		driver:       drv.Name,
		nodeName:     "",
		driverOpts:   drv.Opt,
		flags:        flags,
		bootstrap:    bootstrap,
		platform:     []string{},
		actionAppend: false,
		actionLeave:  false,
		use:          false,
		configFile:   "",
	}, args)
	if err != nil {
		resp.Diagnostics.AddError("Create Instance from options", err.Error())
		return
	}

	d.Id = types.String{
		Value: fmt.Sprintf("%s_%s", dockerCli.DockerEndpoint().Host, name),
	}

	diags = resp.State.Set(ctx, &d)
	resp.Diagnostics.Append(diags...)

	return
}

func (r instanceResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {

	var d instanceResourceData

	diags := req.State.Get(ctx, &d)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Reading Delete Attributes failed", map[string]interface{}{
			"plan": req.State.Raw,
		})
		return
	}

	dockerCli := r.m.Cli

	txn, release := r.getStore(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Getting store failed", map[string]interface{}{
			"plan": req.State.Raw,
		})
		return
	}
	defer release()

	var name string
	if d.Name.Null || d.Name.Value == "" {
		if d.GeneratedName.Null || d.GeneratedName.Value == "" {
			panic("Unexpected null Generated name")
		}
		name = d.GeneratedName.Value
	} else {
		name = d.Name.Value
	}

	drv := instanceDriverData{}
	if d.Driver != nil {
		drv = *d.Driver
	}

	keepState := false
	if !drv.KeepState.Null {
		keepState = drv.KeepState.Value
	}
	err := deleteInstanceByName(ctx, dockerCli, txn, rmOptions{
		builder:   name,
		keepState: keepState,
	})
	if err != nil {
		resp.Diagnostics.AddError("Deleting Instance by name failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r instanceResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var d instanceResourceData

	diags := req.State.Get(ctx, &d)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	txn, release := r.getStore(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	defer release()

	var name string
	if d.Name.Null || d.Name.Value == "" {
		if d.GeneratedName.Null {
			panic("Unexpected empty Generated name")
		}
		name = d.GeneratedName.Value
	} else {
		name = d.Name.Value
	}

	present, err := readInstanceByName(ctx, txn, name, &d)
	if err != nil {
		resp.Diagnostics.AddError("Reading Instance by name failed", err.Error())
		return
	}

	if !present {
		resp.State.RemoveResource(ctx)
	}
}

func (r instanceResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	panic("Update not implemented")
}

func (r instanceResource) getStore(ctx context.Context, diags *diag.Diagnostics) (*store.Txn, func()) {
	tflog.Trace(ctx, "Getting the Store")
	txn, release, err := storeutil.GetStore(r.m.Cli)
	if err != nil {
		(*diags).AddError("Getting Store failed", err.Error())
		return nil, nil
	}
	return txn, release
}
