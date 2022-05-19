package meta

import (
	"github.com/docker/cli/cli/command"
)

type Data struct {
	Cli command.Cli
}

type HasData interface {
	Data() *Data
}
