package grpc

import (
	"io"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	manifeststore "github.com/docker/cli/cli/manifest/store"
	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/cli/trust"
	"github.com/docker/docker/client"
	notaryclient "github.com/theupdateframework/notary/client"
)

type fakeCli struct {
	command.Cli
}

func (c *fakeCli) Client() client.APIClient {
	panic("Not implemented yet")
}

func (c *fakeCli) Out() *streams.Out {
	panic("Not implemented yet")
}

func (c *fakeCli) Err() io.Writer {
	panic("Not implemented yet")
}
func (c *fakeCli) In() *streams.In {
	panic("Not implemented yet")
}
func (c *fakeCli) SetIn(in *streams.In) {
	panic("Not implemented yet")
}
func (c *fakeCli) Apply(ops ...command.DockerCliOption) error {
	panic("Not implemented yet")
}
func (c *fakeCli) ConfigFile() *configfile.ConfigFile {
	return &configfile.ConfigFile{
		Filename: "/etc/docker/daemon.json",
	}
}
func (c *fakeCli) ServerInfo() command.ServerInfo {
	panic("Not implemented yet")
}
func (c *fakeCli) ClientInfo() command.ClientInfo {
	panic("Not implemented yet")
}
func (c *fakeCli) NotaryClient(imgRefAndAuth trust.ImageRefAndAuth, actions []string) (notaryclient.Repository, error) {
	panic("Not implemented yet")
}
func (c *fakeCli) DefaultVersion() string {
	panic("Not implemented yet")
}
func (c *fakeCli) ManifestStore() manifeststore.Store {
	panic("Not implemented yet")
}
func (c *fakeCli) RegistryClient(bool) registryclient.RegistryClient {
	panic("Not implemented yet")
}
func (c *fakeCli) ContentTrustEnabled() bool {
	panic("Not implemented yet")
}
func (c *fakeCli) ContextStore() store.Store {
	panic("Not implemented yet")
}
func (c *fakeCli) CurrentContext() string {
	panic("Not implemented yet")
}
func (c *fakeCli) StackOrchestrator(flagValue string) (command.Orchestrator, error) {
	panic("Not implemented yet")
}
func (c *fakeCli) DockerEndpoint() docker.Endpoint {
	panic("Not implemented yet")
}
