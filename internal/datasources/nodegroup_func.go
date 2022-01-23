package datasources

import (
	"context"
	"fmt"
	"os"

	"github.com/abergmeier/terraform-provider-buildx/pkg/buildx/commands"
	"github.com/docker/buildx/store/storeutil"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func readNodeGroup(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	md, err := readNodeGroupByName(ctx, nil, name)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("endpoints", md.Endpoints)
	return nil
}

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

	fmt.Fprintf(os.Stderr, "EP %#v", md.Endpoints)
	return md, nil
}
