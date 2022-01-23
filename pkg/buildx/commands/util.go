package commands

import (
	"context"
	"net/url"
	"os"
	"strings"

	"github.com/docker/buildx/build"
	"github.com/docker/buildx/driver"
	"github.com/docker/buildx/store"
	"github.com/docker/buildx/store/storeutil"
	"github.com/docker/buildx/util/platformutil"
	"github.com/docker/buildx/util/progress"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	ctxstore "github.com/docker/cli/cli/context/store"
	dopts "github.com/docker/cli/opts"
	dockerclient "github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/tools/clientcmd"
)

// ValidateEndpoint validates that endpoint is either a context or a docker host
func ValidateEndpoint(dockerCli command.Cli, ep string) (string, error) {
	de, err := storeutil.GetDockerEndpoint(dockerCli, ep)
	if err == nil && de != "" {
		if ep == "default" {
			return de, nil
		}
		return ep, nil
	}
	h, err := dopts.ParseHost(true, ep)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse endpoint %s", ep)
	}
	return h, nil
}

// driversForNodeGroup returns drivers for a nodegroup instance
func driversForNodeGroup(ctx context.Context, dockerCli command.Cli, ng *store.NodeGroup, contextPathHash string) ([]build.DriverInfo, error) {
	eg, _ := errgroup.WithContext(ctx)

	dis := make([]build.DriverInfo, len(ng.Nodes))

	var f driver.Factory
	if ng.Driver != "" {
		f = driver.GetFactory(ng.Driver, true)
		if f == nil {
			return nil, errors.Errorf("failed to find driver %q", f)
		}
	} else {
		dockerapi, err := clientForEndpoint(dockerCli, ng.Nodes[0].Endpoint)
		if err != nil {
			return nil, err
		}
		f, err = driver.GetDefaultFactory(ctx, dockerapi, false)
		if err != nil {
			return nil, err
		}
		ng.Driver = f.Name()
	}
	imageopt, err := storeutil.GetImageConfig(dockerCli, ng)
	if err != nil {
		return nil, err
	}

	for i, n := range ng.Nodes {
		func(i int, n store.Node) {
			eg.Go(func() error {
				di := build.DriverInfo{
					Name:     n.Name,
					Platform: n.Platforms,
				}
				defer func() {
					dis[i] = di
				}()
				dockerapi, err := clientForEndpoint(dockerCli, n.Endpoint)
				if err != nil {
					di.Err = err
					return nil
				}
				// TODO: replace the following line with dockerclient.WithAPIVersionNegotiation option in clientForEndpoint
				dockerapi.NegotiateAPIVersion(ctx)

				contextStore := dockerCli.ContextStore()

				var kcc driver.KubeClientConfig
				kcc, err = configFromContext(n.Endpoint, contextStore)
				if err != nil {
					// err is returned if n.Endpoint is non-context name like "unix:///var/run/docker.sock".
					// try again with name="default".
					// FIXME: n should retain real context name.
					kcc, err = configFromContext("default", contextStore)
					if err != nil {
						logrus.Error(err)
					}
				}

				tryToUseKubeConfigInCluster := false
				if kcc == nil {
					tryToUseKubeConfigInCluster = true
				} else {
					if _, err := kcc.ClientConfig(); err != nil {
						tryToUseKubeConfigInCluster = true
					}
				}
				if tryToUseKubeConfigInCluster {
					kccInCluster := driver.KubeClientConfigInCluster{}
					if _, err := kccInCluster.ClientConfig(); err == nil {
						logrus.Debug("using kube config in cluster")
						kcc = kccInCluster
					}
				}

				d, err := driver.GetDriver(ctx, "buildx_buildkit_"+n.Name, f, dockerapi, imageopt.Auth, kcc, n.Flags, n.Files, n.DriverOpts, n.Platforms, contextPathHash)
				if err != nil {
					di.Err = err
					return nil
				}
				di.Driver = d
				di.ImageOpt = imageopt
				return nil
			})
		}(i, n)
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return dis, nil
}

func configFromContext(endpointName string, s ctxstore.Reader) (clientcmd.ClientConfig, error) {
	if strings.HasPrefix(endpointName, "kubernetes://") {
		u, _ := url.Parse(endpointName)
		if kubeconfig := u.Query().Get("kubeconfig"); kubeconfig != "" {
			_ = os.Setenv(clientcmd.RecommendedConfigPathEnvVar, kubeconfig)
		}
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		apiConfig, err := rules.Load()
		if err != nil {
			return nil, err
		}
		return clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{}), nil
	}
	return kubernetes.ConfigFromContext(endpointName, s)
}

// clientForEndpoint returns a docker client for an endpoint
func clientForEndpoint(dockerCli command.Cli, name string) (dockerclient.APIClient, error) {
	list, err := dockerCli.ContextStore().List()
	if err != nil {
		return nil, err
	}
	for _, l := range list {
		if l.Name == name {
			dep, ok := l.Endpoints["docker"]
			if !ok {
				return nil, errors.Errorf("context %q does not have a Docker endpoint", name)
			}
			epm, ok := dep.(docker.EndpointMeta)
			if !ok {
				return nil, errors.Errorf("endpoint %q is not of type EndpointMeta, %T", dep, dep)
			}
			ep, err := docker.WithTLSData(dockerCli.ContextStore(), name, epm)
			if err != nil {
				return nil, err
			}
			clientOpts, err := ep.ClientOpts()
			if err != nil {
				return nil, err
			}
			return dockerclient.NewClientWithOpts(clientOpts...)
		}
	}

	ep := docker.Endpoint{
		EndpointMeta: docker.EndpointMeta{
			Host: name,
		},
	}

	clientOpts, err := ep.ClientOpts()
	if err != nil {
		return nil, err
	}

	return dockerclient.NewClientWithOpts(clientOpts...)
}

func GetInstanceOrDefault(ctx context.Context, dockerCli command.Cli, instance, contextPathHash string) ([]build.DriverInfo, error) {
	var defaultOnly bool

	if instance == "default" && instance != dockerCli.CurrentContext() {
		return nil, errors.Errorf("use `docker --context=default buildx` to switch to default context")
	}
	if instance == "default" || instance == dockerCli.CurrentContext() {
		instance = ""
		defaultOnly = true
	}
	list, err := dockerCli.ContextStore().List()
	if err != nil {
		return nil, err
	}
	for _, l := range list {
		if l.Name == instance {
			return nil, errors.Errorf("use `docker --context=%s buildx` to switch to context %s", instance, instance)
		}
	}

	if instance != "" {
		return getInstanceByName(ctx, dockerCli, instance, contextPathHash)
	}
	return getDefaultDrivers(ctx, dockerCli, defaultOnly, contextPathHash)
}

func getInstanceByName(ctx context.Context, dockerCli command.Cli, instance, contextPathHash string) ([]build.DriverInfo, error) {
	txn, release, err := storeutil.GetStore(dockerCli)
	if err != nil {
		return nil, err
	}
	defer release()

	ng, err := txn.NodeGroupByName(instance)
	if err != nil {
		return nil, err
	}
	return driversForNodeGroup(ctx, dockerCli, ng, contextPathHash)
}

// getDefaultDrivers returns drivers based on current cli config
func getDefaultDrivers(ctx context.Context, dockerCli command.Cli, defaultOnly bool, contextPathHash string) ([]build.DriverInfo, error) {
	txn, release, err := storeutil.GetStore(dockerCli)
	if err != nil {
		return nil, err
	}
	defer release()

	if !defaultOnly {
		ng, err := storeutil.GetCurrentInstance(txn, dockerCli)
		if err != nil {
			return nil, err
		}

		if ng != nil {
			return driversForNodeGroup(ctx, dockerCli, ng, contextPathHash)
		}
	}

	imageopt, err := storeutil.GetImageConfig(dockerCli, nil)
	if err != nil {
		return nil, err
	}

	d, err := driver.GetDriver(ctx, "buildx_buildkit_default", nil, dockerCli.Client(), imageopt.Auth, nil, nil, nil, nil, nil, contextPathHash)
	if err != nil {
		return nil, err
	}
	return []build.DriverInfo{
		{
			Name:     "default",
			Driver:   d,
			ImageOpt: imageopt,
		},
	}, nil
}

func loadInfoData(ctx context.Context, d *dinfo) error {
	if d.di.Driver == nil {
		return nil
	}
	info, err := d.di.Driver.Info(ctx)
	if err != nil {
		return err
	}
	d.info = info
	if info.Status == driver.Running {
		c, err := d.di.Driver.Client(ctx)
		if err != nil {
			return err
		}
		workers, err := c.ListWorkers(ctx)
		if err != nil {
			return errors.Wrap(err, "listing workers")
		}
		for _, w := range workers {
			for _, p := range w.Platforms {
				d.platforms = append(d.platforms, p)
			}
		}
		d.platforms = platformutil.Dedupe(d.platforms)
	}
	return nil
}

func LoadNodeGroupData(ctx context.Context, dockerCli command.Cli, ngi *Nginfo) error {
	eg, _ := errgroup.WithContext(ctx)

	dis, err := driversForNodeGroup(ctx, dockerCli, ngi.Ng, "")
	if err != nil {
		return err
	}
	ngi.drivers = make([]dinfo, len(dis))
	for i, di := range dis {
		d := di
		ngi.drivers[i].di = &d
		func(d *dinfo) {
			eg.Go(func() error {
				if err := loadInfoData(ctx, d); err != nil {
					d.err = err
				}
				return nil
			})
		}(&ngi.drivers[i])
	}

	if eg.Wait(); err != nil {
		return err
	}

	kubernetesDriverCount := 0

	for _, di := range ngi.drivers {
		if di.info != nil && len(di.info.DynamicNodes) > 0 {
			kubernetesDriverCount++
		}
	}

	isAllKubernetesDrivers := len(ngi.drivers) == kubernetesDriverCount

	if isAllKubernetesDrivers {
		var drivers []dinfo
		var dynamicNodes []store.Node

		for _, di := range ngi.drivers {
			// dynamic nodes are used in Kubernetes driver.
			// Kubernetes pods are dynamically mapped to BuildKit Nodes.
			if di.info != nil && len(di.info.DynamicNodes) > 0 {
				for i := 0; i < len(di.info.DynamicNodes); i++ {
					// all []dinfo share *build.DriverInfo and *driver.Info
					diClone := di
					if pl := di.info.DynamicNodes[i].Platforms; len(pl) > 0 {
						diClone.platforms = pl
					}
					drivers = append(drivers, di)
				}
				dynamicNodes = append(dynamicNodes, di.info.DynamicNodes...)
			}
		}

		// not append (remove the static nodes in the store)
		ngi.Ng.Nodes = dynamicNodes
		ngi.drivers = drivers
		ngi.Ng.Dynamic = true
	}

	return nil
}

func dockerAPI(dockerCli command.Cli) *api {
	return &api{dockerCli: dockerCli}
}

type api struct {
	dockerCli command.Cli
}

func (a *api) DockerAPI(name string) (dockerclient.APIClient, error) {
	if name == "" {
		name = a.dockerCli.CurrentContext()
	}
	return clientForEndpoint(a.dockerCli, name)
}

type dinfo struct {
	di        *build.DriverInfo
	info      *driver.Info
	platforms []specs.Platform
	err       error
}

type Nginfo struct {
	Ng      *store.NodeGroup
	drivers []dinfo
	Err     error
}

func Boot(ctx context.Context, ngi *Nginfo) (bool, error) {
	toBoot := make([]int, 0, len(ngi.drivers))
	for i, d := range ngi.drivers {
		if d.err != nil || d.di.Err != nil || d.di.Driver == nil || d.info == nil {
			continue
		}
		if d.info.Status != driver.Running {
			toBoot = append(toBoot, i)
		}
	}
	if len(toBoot) == 0 {
		return false, nil
	}

	printer := progress.NewPrinter(context.TODO(), os.Stderr, "auto")

	baseCtx := ctx
	eg, _ := errgroup.WithContext(ctx)
	for _, idx := range toBoot {
		func(idx int) {
			eg.Go(func() error {
				pw := progress.WithPrefix(printer, ngi.Ng.Nodes[idx].Name, len(toBoot) > 1)
				_, err := driver.Boot(ctx, baseCtx, ngi.drivers[idx].di.Driver, pw)
				if err != nil {
					ngi.drivers[idx].err = err
				}
				return nil
			})
		}(idx)
	}

	err := eg.Wait()
	err1 := printer.Wait()
	if err == nil {
		err = err1
	}

	return true, err
}
