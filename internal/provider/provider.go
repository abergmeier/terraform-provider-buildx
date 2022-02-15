package provider

import (
	"context"
	"log"
	"net"

	"github.com/abergmeier/terraform-provider-buildx/internal/datasources"
	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/abergmeier/terraform-provider-buildx/internal/resources"
	"github.com/abergmeier/terraform-provider-buildx/pkg/grpc"
	"github.com/docker/cli/cli/command"

	cliflags "github.com/docker/cli/cli/flags"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	// Register drivers
	_ "github.com/docker/buildx/driver/docker"
	_ "github.com/docker/buildx/driver/docker-container"
	_ "github.com/docker/buildx/driver/kubernetes"
)

func Provider() *schema.Provider {
	provider := &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"buildx_booted":    datasources.BootedDataSource(),
			"buildx_nodegroup": datasources.NodeGroupDataSource(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"buildx_built":    resources.BuiltResource(),
			"buildx_instance": resources.InstanceResource(),
		},
		Schema: map[string]*schema.Schema{},
	}
	provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		// Shameless plug from https://github.com/terraform-providers/terraform-provider-aws/blob/d51784148586f605ab30ecea268e80fe83d415a9/aws/provider.go
		terraformVersion := provider.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}
		return providerConfigure(ctx, d, terraformVersion)
	}
	return provider
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {

	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, diag.FromErr(err)
	}
	opts := cliflags.NewClientOptions()
	err = dockerCli.Initialize(opts)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	m := &meta.Data{
		Cli:      dockerCli,
		GRPCAddr: *lis.Addr().(*net.TCPAddr),
	}

	go grpc.Serve(lis)

	return m, nil
}
