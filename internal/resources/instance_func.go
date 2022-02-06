package resources

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/abergmeier/terraform-provider-buildx/internal/consolefile"
	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/docker/buildx/commands"
	"github.com/docker/buildx/driver"
	"github.com/docker/buildx/store"
	"github.com/docker/buildx/store/storeutil"
	"github.com/docker/buildx/util/progress"
	"github.com/docker/cli/cli/command"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
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

func createInstance(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	dockerCli := m.(*meta.Data).Cli
	tflog.Trace(ctx, "Getting the Store")
	txn, release, err := storeutil.GetStore(dockerCli)
	if err != nil {
		return diag.FromErr(err)
	}
	defer release()

	name, err := handleName(d, txn)
	if err != nil {
		return diag.FromErr(err)
	}

	args := []string{}
	dockerContext := d.Get("context").(string)
	dockerEndpoint := d.Get("endpoint").(string)
	if dockerContext != "" {
		args = append(args, dockerContext)
	} else if dockerEndpoint != "" {
		args = append(args, dockerEndpoint)
	}

	var flags []string
	buildkits := d.Get("buildkit").(*schema.Set).List()
	if len(buildkits) != 0 {
		buildkit := buildkits[0].(map[string]interface{})

		fi := buildkit["flags"].([]interface{})
		flags = make([]string, len(fi))
		for i, f := range fi {
			flags[i] = f.(string)
		}
	}

	driver := d.Get("driver").(*schema.Set).List()[0].(map[string]interface{})
	oi := driver["opt"].(map[string]interface{})
	opts := make(map[string]string, len(oi))
	for k, v := range oi {
		opts[k] = v.(string)
	}

	err = createInstanceFromOptions(ctx, dockerCli, txn, createOptions{
		name:         name,
		driver:       driver["name"].(string),
		nodeName:     "",
		driverOpts:   opts,
		flags:        flags,
		bootstrap:    d.Get("bootstrap").(bool),
		platform:     []string{},
		actionAppend: false,
		actionLeave:  false,
		use:          false,
		configFile:   "",
	}, args)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%s_%s", dockerCli.DockerEndpoint().Host, name))
	return nil
}

func handleName(d *schema.ResourceData, txn *store.Txn) (name string, err error) {
	if d.Get("generate_name").(bool) {
		name, err = store.GenerateName(txn)
		if err != nil {
			return "", err
		}
		if name == "" {
			panic("GenerateName call returned empty name!")
		}
		err = d.Set("generated_name", name)
		if err != nil {
			return "", err
		}
		err = d.Set("name", nil)
		if err != nil {
			return "", err
		}
	} else {
		err = d.Set("generated_name", nil)
		if err != nil {
			return "", err
		}
		name = d.Get("name").(string)
		if name == "" {
			return "", errors.New("Either generate_name needs to be set to true or a name needs to be specified.")
		}
	}

	return
}

func createInstanceFromOptions(ctx context.Context, dockerCli command.Cli, txn *store.Txn, in createOptions, args []string) error {

	driverName := in.driver

	ctx = tflog.With(ctx, "driver.name", driverName)
	tflog.Trace(ctx, "Get Factory for Driver")
	if driver.GetFactory(driverName, true) == nil {
		return errors.Errorf("failed to find driver %q", in.driver)
	}

	name := in.name
	ctx = tflog.With(ctx, "nodegroup.name", name)

	tflog.Trace(ctx, "Get NodeGroup")
	ng, err := txn.NodeGroupByName(name)
	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
		} else {
			return err
		}
	}

	if ng != nil {
		return errors.Errorf("existing instance for %s but no append mode, specify --node to make changes for existing instances", name)
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
			return errors.Errorf("could not create a builder instance with TLS data loaded from environment. Please use `docker context create <context-name>` to create a context for current environment and then create a builder instance with `docker buildx create <context-name>`")
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

func deleteInstance(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	dockerCli := m.(*meta.Data).Cli
	tflog.Trace(ctx, "Getting the Store")
	txn, release, err := storeutil.GetStore(dockerCli)
	if err != nil {
		return diag.FromErr(err)
	}
	defer release()
	name := d.Get("name").(string)
	if name == "" {
		name = d.Get("generated_name").(string)
	}

	buildkits := d.Get("buildkit").(*schema.Set).List()
	keepState := false
	if len(buildkits) != 0 {
		buildkit := buildkits[0].(map[string]interface{})
		keepState = buildkit["keep_state"].(bool)
	}

	err = deleteInstanceByName(ctx, dockerCli, txn, rmOptions{
		builder:   name,
		keepState: keepState,
	})
	return diag.FromErr(err)
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

func readInstance(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	txn, release, err := storeutil.GetStore(m.(*meta.Data).Cli)
	if err != nil {
		return diag.FromErr(err)
	}
	defer release()
	name := d.Get("name").(string)
	if name == "" {
		name = d.Get("generated_name").(string)
	}
	err = readInstanceByName(ctx, txn, name, d)
	return diag.FromErr(err)
}

func readInstanceByName(ctx context.Context, txn *store.Txn, name string, d *schema.ResourceData) error {

	tflog.Trace(ctx, "List")
	ll, err := txn.List()
	if err != nil {
		return err
	}

	ctx = tflog.With(ctx, "nodegroup.name", name)
	tflog.Trace(ctx, "Searching NodeGroup")
	for _, ng := range ll {
		tflog.Trace(ctx, "Found NodeGroup")
		if ng.Name == name {
			// Found
			return nil
		}
	}

	tflog.Trace(ctx, "NodeGroup not found")
	// Seems like instance is gone
	d.SetId("")
	return nil
}
