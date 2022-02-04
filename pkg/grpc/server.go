package grpc

import (
	"context"
	"net"

	"github.com/abergmeier/terraform-provider-buildx/pkg/grpc/api"
	"github.com/docker/buildx/commands"
	"github.com/docker/buildx/driver"
	"github.com/docker/cli/cli/command"
	"google.golang.org/grpc"
)

func Serve(l net.Listener) error {
	var opts []grpc.ServerOption

	cli, err := command.NewDockerCli()
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(opts...)
	api.RegisterDriversServer(grpcServer, &driversServer{
		cli: cli,
	})
	api.RegisterBuildxServer(grpcServer, &buildXServer{
		cli: cli,
	})
	return grpcServer.Serve(l)
}

type buildXServer struct {
	api.UnimplementedBuildxServer

	cli command.Cli
}

type driversServer struct {
	api.UnimplementedDriversServer

	cli command.Cli
}

func (s *buildXServer) BootByInstanceName(ctx context.Context, req *api.InstanceByNameRequest) (*api.BootByInstanceNameResponse, error) {
	di, err := commands.GetInstanceOrDefault(ctx, s.cli, req.Instance, req.ContextPathHash)
	if err != nil {
		return nil, err
	}

	client, err := driver.Boot(ctx, ctx, di[0].Driver, nil)
	if err != nil {
		return nil, err
	}
	return &api.BootByInstanceNameResponse{}, client.Close()
}

func (s *driversServer) InstanceOrDefaultByName(ctx context.Context, req *api.InstanceByNameRequest) (*api.InstanceByNameResponse, error) {
	di, err := commands.GetInstanceOrDefault(ctx, s.cli, req.Instance, req.ContextPathHash)
	if err != nil {
		return nil, err
	}

	dis := make([]*api.DriverInfo, 0, len(di))
	for _, d := range di {
		dis = append(dis, &api.DriverInfo{
			Name: d.Name,
		})
	}
	return &api.InstanceByNameResponse{
		DriverInfos: dis,
	}, nil
}
