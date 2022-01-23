package grpc

import (
	"context"
	"net"
	"reflect"
	"testing"

	"github.com/abergmeier/terraform-provider-buildx/pkg/grpc/api"
	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	// Register drivers
	_ "github.com/docker/buildx/driver/docker"
	_ "github.com/docker/buildx/driver/docker-container"
	_ "github.com/docker/buildx/driver/kubernetes"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestDriversServer(t *testing.T) {

	dockerCli, err := command.NewDockerCli()
	if err != nil {
		t.Fatal(err)
	}
	opts := cliflags.NewClientOptions()
	err = dockerCli.Initialize(opts)
	if err != nil {
		t.Fatal(err)
	}

	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	api.RegisterDriversServer(s, &driversServer{
		cli: dockerCli,
	})
	go func() {
		err := s.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := api.NewDriversClient(conn)
	resp, err := client.InstanceOrDefaultByName(ctx, &api.InstanceByNameRequest{
		Instance:        "default",
		ContextPathHash: "/foo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.DriverInfos) == 0 {
		t.Fatal("Expected DriverInfos")
	}

	if !reflect.DeepEqual(resp.DriverInfos, []*api.DriverInfo{
		{
			Name: "default",
		},
	}) {
		t.Fatal("Expected changes in DriverInfos", resp.DriverInfos)
	}
}

func TestBuildxServer(t *testing.T) {

	dockerCli, err := command.NewDockerCli()
	if err != nil {
		t.Fatal(err)
	}
	opts := cliflags.NewClientOptions()
	err = dockerCli.Initialize(opts)
	if err != nil {
		t.Fatal(err)
	}

	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	api.RegisterBuildxServer(s, &buildXServer{
		cli: dockerCli,
	})
	go func() {
		err := s.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := api.NewBuildxClient(conn)
	resp, err := client.BootByInstanceName(ctx, &api.InstanceByNameRequest{
		Instance:        "default",
		ContextPathHash: "/foo",
	})
	if err != nil {
		t.Fatal("BootByInstanceName failed:", err)
	}
	if resp == nil {
		t.Fatal("Resp is nil")
	}
}
