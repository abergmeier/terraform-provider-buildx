package commands

import (
	"context"

	"github.com/docker/buildx/store"
	"github.com/docker/cli/cli/command"
)

func Rm(ctx context.Context, dockerCli command.Cli, ng *store.NodeGroup, keepState bool) error {
	dis, err := driversForNodeGroup(ctx, dockerCli, ng, "")
	if err != nil {
		return err
	}
	for _, di := range dis {
		if di.Driver != nil {
			if err := di.Driver.Stop(ctx, true); err != nil {
				return err
			}
			if err := di.Driver.Rm(ctx, true, !keepState); err != nil {
				return err
			}
		}
		if di.Err != nil {
			err = di.Err
		}
	}
	return err
}
