package datasources

import (
	"context"
	"fmt"

	"github.com/docker/buildx/commands"
	"github.com/docker/buildx/store/storeutil"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
)

func readNodeGroupByName(ctx context.Context, dockerCli command.Cli, name string) (*store.Metadata, error) {
	txn, release, err := storeutil.GetStore(dockerCli)
	if err != nil {
		return nil, err
	}
	defer release()

	ll, err := txn.List()
	if err != nil {
		return nil, err
	}

	builders := make([]*commands.Nginfo, len(ll))
	for i, ng := range ll {
		builders[i] = &commands.Nginfo{Ng: ng}
	}

	list, err := dockerCli.ContextStore().List()
	if err != nil {
		return nil, err
	}

	var md *store.Metadata
	for _, l := range list {
		if l.Name == name {
			md = &l
			break
		}
	}

	if md == nil {
		return nil, fmt.Errorf("no NodeGroup with name %s found", name)
	}

	return md, nil
}
