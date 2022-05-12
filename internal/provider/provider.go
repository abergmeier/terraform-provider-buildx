package provider

import (
	"context"

	"github.com/abergmeier/terraform-provider-buildx/internal/datasources"
	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/abergmeier/terraform-provider-buildx/internal/resources"
	"github.com/docker/cli/cli/command"

	cliflags "github.com/docker/cli/cli/flags"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	// Register drivers
	_ "github.com/docker/buildx/driver/docker"
	_ "github.com/docker/buildx/driver/docker-container"
	_ "github.com/docker/buildx/driver/kubernetes"
)

type provider struct {
	data *meta.Data

	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type providerConfig struct {
}

func New(version string) func() tfsdk.Provider {
	return func() tfsdk.Provider {
		return &provider{
			version: version,
		}
	}
}

func (p *provider) Data() *meta.Data {
	return p.data
}

func (p *provider) Configure(ctx context.Context, req tfsdk.ConfigureProviderRequest, resp *tfsdk.ConfigureProviderResponse) {
	var config providerConfig
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	dockerCli, err := command.NewDockerCli()
	if err != nil {
		resp.Diagnostics.AddError("Creating Docker Cli failed", err.Error())
		return
	}
	opts := cliflags.NewClientOptions()
	err = dockerCli.Initialize(opts)
	if err != nil {
		resp.Diagnostics.AddError("Initializing Docker Cli failed", err.Error())
		return
	}

	p.data = &meta.Data{
		Cli: dockerCli,
	}
}

func (p *provider) GetResources(ctx context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"buildx_built": resources.BuiltType,
		// We use instance as a resource so we can import already present
		// instances
		"buildx_instance": resources.InstanceType,
	}, nil
}

func (p *provider) GetDataSources(ctx context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"buildx_nodegroup": datasources.NodeGroupType,
	}, nil
}

func (p *provider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{}, nil
}
