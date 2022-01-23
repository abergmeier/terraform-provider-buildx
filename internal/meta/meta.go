package meta

import (
	"net"

	"github.com/docker/cli/cli/command"
)

type Data struct {
	Cli      command.Cli
	GRPCAddr net.TCPAddr
}
