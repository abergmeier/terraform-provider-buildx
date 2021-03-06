package resources

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/abergmeier/terraform-provider-buildx/internal/consolefile"
	"github.com/docker/buildx/commands"
	"github.com/docker/buildx/driver"
	"github.com/docker/buildx/store"
	"github.com/docker/buildx/store/storeutil"
	"github.com/docker/buildx/util/progress"
	"github.com/docker/cli/cli/command"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type createOptions struct {
	name         string
	driver       string
	nodeName     string
	platform     []string
	actionAppend bool
	actionLeave  bool
	use          bool
	flags        []string
	configFile   string
	driverOpts   map[string]string
	bootstrap    bool
	// upgrade      bool // perform upgrade of the driver
}

type rmOptions struct {
	builder   string
	keepState bool
}

func handleNameAttributes(d *instanceResourceData, txn *store.Txn) (name string, err error) {
	if !d.GenerateName.Null && d.GenerateName.Value {
		name, err = store.GenerateName(txn)
		if err != nil {
			return "", err
		}
		if name == "" {
			panic("GenerateName call returned empty name!")
		}
		d.GeneratedName = types.String{
			Value: name,
		}
	} else {
		if d.Name.Null || d.Name.Value == "" {
			return "", errors.New("Either generate_name needs to be set to true or a name needs to be specified.")
		}
		d.GeneratedName = types.String{
			Null: true,
		}
		name = d.Name.Value
	}

	return
}

func createInstanceFromOptions(ctx context.Context, dockerCli command.Cli, txn *store.Txn, in createOptions, args []string) error {

	driverName := in.driver

	ctx = tflog.With(ctx, "driver.name", driverName)
	tflog.Trace(ctx, "Get Factory for Driver")
	if driver.GetFactory(driverName, true) == nil {
		return fmt.Errorf("failed to find driver %q", in.driver)
	}

	name := in.name
	ctx = tflog.With(ctx, "nodegroup.name", name)

	tflog.Trace(ctx, "Get NodeGroup")
	ng, err := txn.NodeGroupByName(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
		} else {
			tflog.Error(ctx, "NodGroupByName failed", map[string]interface{}{
				"name": name,
			})
			return err
		}
	}

	if ng != nil {
		return fmt.Errorf("existing instance for %s but no append mode, specify --node to make changes for existing instances", name)
	}

	ng = &store.NodeGroup{
		Name: name,
	}

	if ng.Driver == "" || in.driver != "" {
		ng.Driver = driverName
	}

	flags := in.flags

	var ep string

	if len(args) > 0 {
		ctx = tflog.With(ctx, "endpoint", args[0])
		tflog.Trace(ctx, "Validate Endpoint")
		ep, err = commands.ValidateEndpoint(dockerCli, args[0])
		if err != nil {
			return err
		}
	} else {
		if dockerCli.CurrentContext() == "default" && dockerCli.DockerEndpoint().TLSData != nil {
			return errors.New("could not create a builder instance with TLS data loaded from environment. Please use `docker context create <context-name>` to create a context for current environment and then create a builder instance with `docker buildx create <context-name>`")
		}

		tflog.Trace(ctx, "Get Endpoint")
		ep, err = storeutil.GetCurrentEndpoint(dockerCli)
		if err != nil {
			return err
		}
	}

	if in.driver == "kubernetes" {
		// naming endpoint to make --append works
		ep = (&url.URL{
			Scheme: in.driver,
			Path:   "/" + in.name,
			RawQuery: (&url.Values{
				"deployment": {""},
				"kubeconfig": {os.Getenv("KUBECONFIG")},
			}).Encode(),
		}).String()
	}

	m := in.driverOpts
	tflog.Trace(ctx, "Update NodeGroup")
	if err := ng.Update("", ep, in.platform, len(args) > 0, false, flags, in.configFile, m); err != nil {
		return err
	}

	tflog.Trace(ctx, "Save Transaction")
	if err := txn.Save(ng); err != nil {
		return err
	}

	ngi := &commands.Nginfo{Ng: ng}

	timeoutCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	tflog.Trace(ctx, "Load NodeGroup Data")
	if err = commands.LoadNodeGroupData(timeoutCtx, dockerCli, ngi); err != nil {
		return err
	}

	if in.bootstrap {

		cf := consolefile.WithPrefix(ctx, os.Stderr, tflog.Info)
		printer := progress.NewPrinter(context.TODO(), cf, "auto")
		if _, err = commands.BootWithWriter(ctx, ngi, func(prefix string, force bool) progress.Writer {
			return progress.WithPrefix(printer, prefix, true)
		}); err != nil {
			return err
		}
		if err := printer.Wait(); err != nil {
			return err
		}
	}

	return nil
}

func deleteInstanceByName(ctx context.Context, dockerCli command.Cli, txn *store.Txn, in rmOptions) error {

	ctx = tflog.With(ctx, "nodegroup.name", in.builder)
	tflog.Trace(ctx, "Get NodeGroup")
	ng, err := storeutil.GetNodeGroup(txn, dockerCli, in.builder)
	if err != nil {
		return err
	}
	ctx = tflog.With(ctx, "keep_state", in.keepState)
	tflog.Trace(ctx, "Rm")
	err1 := commands.Rm(ctx, dockerCli, ng, in.keepState)
	if err := txn.Remove(ng.Name); err != nil {
		return err
	}

	return err1
}

func readInstanceByName(ctx context.Context, txn *store.Txn, name string, d *instanceResourceData) (bool, error) {

	tflog.Trace(ctx, "List")
	ll, err := txn.List()
	if err != nil {
		return false, fmt.Errorf("Listing nodes failed: %w", err)
	}

	ctx = tflog.With(ctx, "nodegroup.name", name)
	tflog.Trace(ctx, "Searching NodeGroup")
	for _, ng := range ll {
		tflog.Trace(ctx, "Found NodeGroup")
		if ng.Name == name {
			// Found
			return true, nil
		}
	}

	tflog.Trace(ctx, "NodeGroup not found")
	// Seems like instance is gone
	return false, nil
}

func atLeastOneOf(ln string, lf func() bool, rn string, rf func() bool) diag.Diagnostics {
	l := lf()
	r := rf()

	if l || r {
		return nil
	}

	allKeys := []string{
		ln, rn,
	}
	return diag.Diagnostics{
		diag.NewErrorDiagnostic("Missing attributes", fmt.Sprintf("one of `%s` must be specified", strings.Join(allKeys, ","))),
	}
}

func exactlyOneOf(ln string, lf func() bool, rn string, rf func() bool) diag.Diagnostics {
	diags := conflictsWith(ln, lf, rn, rf)
	if diags.HasError() {
		return diags
	}

	diags = atLeastOneOf(ln, lf, rn, rf)
	return diags
}

func conflictsWithString(ln string, lv string, rn string, rv string) diag.Diagnostics {
	if lv == "" || rv == "" {
		return nil
	}

	allKeys := []string{
		ln, rn,
	}
	specified := allKeys
	return diag.Diagnostics{
		diag.NewErrorDiagnostic("Conflicting attributes", fmt.Sprintf("only one of `%s` can be specified, but `%s` were specified.", strings.Join(allKeys, ","), strings.Join(specified, ","))),
	}
}

func conflictsWith(ln string, lf func() bool, rn string, rf func() bool) diag.Diagnostics {
	l := lf()
	r := rf()
	if !(l && r) {
		return nil
	}

	allKeys := []string{
		ln, rn,
	}
	specified := allKeys
	return diag.Diagnostics{
		diag.NewErrorDiagnostic("Conflicting attributes", fmt.Sprintf("only one of `%s` can be specified, but `%s` were specified.", strings.Join(allKeys, ","), strings.Join(specified, ","))),
	}
}
