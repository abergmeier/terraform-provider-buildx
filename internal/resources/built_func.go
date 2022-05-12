package resources

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/abergmeier/terraform-provider-buildx/internal/consolefile"
	"github.com/abergmeier/terraform-provider-buildx/internal/exportentry"
	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/transports"
	_ "github.com/containers/image/v5/transports/alltransports"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/docker/buildx/build"
	"github.com/docker/buildx/commands"
	"github.com/docker/buildx/util/buildflags"
	"github.com/docker/buildx/util/platformutil"
	"github.com/docker/buildx/util/tracing"
	"github.com/docker/cli/cli/command"
	dockeropts "github.com/docker/cli/opts"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/pkg/errors"
)

type buildOptions struct {
	contextPath    string
	dockerfileName string

	allow        []string
	buildArgs    map[string]string
	cacheFrom    []client.CacheOptionsEntry
	cacheTo      []client.CacheOptionsEntry
	cgroupParent string
	extraHosts   []string
	imageIDFile  string
	labels       map[string]string
	networkMode  string
	outputs      []exportentry.TypedEntry
	platforms    []string
	secrets      []string
	shmSize      dockeropts.MemBytes
	ssh          []string
	tags         []string
	target       string
	ulimits      *dockeropts.UlimitOpt
	commonOptions
}

type outputOptions struct {
	Type       string
	attrs      map[string]string
	output_dir string
}

type commonOptions struct {
	builder      string
	metadataFile string
	noCache      *bool
	progress     string
	pull         *bool

	// Not supported by provider
	// exportPush bool
	// exportLoad bool
}

func toCacheEntry(ce []cacheEntryData) []client.CacheOptionsEntry {
	ces := make([]client.CacheOptionsEntry, len(ce))
	for i, e := range ce {
		if e.Type.Null {
			panic(e.Type.Value)
		}
		ces[i] = client.CacheOptionsEntry{
			Type:  e.Type.Value,
			Attrs: e.Attrs,
		}
	}
	return ces
}

func toOutputOptions(ds *builtOutputData) ([]exportentry.TypedEntry, error) {
	if ds == nil {
		return nil, nil
	}

	var outs []exportentry.TypedEntry

	out := exportentry.TypedEntry{}
	if ds.Docker != nil {
		out.Context = ds.Docker.Context
		out.Dest = ds.Docker.Dest
		out.Type = client.ExporterDocker
		outs = append(outs, out)
	}

	if ds.Image != nil {
		out = exportentry.TypedEntry{
			Entry: exportentry.Entry{
				Name: ds.Image.Name,
				// We intentionally do not support pushing
			},
			Type: client.ExporterImage,
		}
		outs = append(outs, out)
	}

	if ds.Local != nil {
		out = exportentry.TypedEntry{
			Entry: exportentry.Entry{
				Dest: ds.Local.Dest,
			},
			Type: client.ExporterLocal,
		}
		outs = append(outs, out)
	}

	if ds.OCI != nil {
		out = exportentry.TypedEntry{
			Entry: exportentry.Entry{
				Dest: ds.OCI.Dest,
			},
			Type: client.ExporterOCI,
		}
		outs = append(outs, out)
	}

	if ds.Tar != nil {
		if ds.Tar.Dest.Null || ds.Tar.Dest.Value == "" {
			return nil, fmt.Errorf("Type %s needs argument dest set", client.ExporterTar)
		}

		out = exportentry.TypedEntry{
			Entry: exportentry.Entry{
				Dest: ds.Tar.Dest,
			},
			Type: client.ExporterTar,
		}
		outs = append(outs, out)
	}

	return outs, nil
}

func toStringMap(mi map[string]interface{}) map[string]string {
	m := make(map[string]string, len(mi))
	for k, v := range mi {
		m[k] = v.(string)
	}
	return m
}

func toStringSlice(l []interface{}) []string {
	sl := make([]string, len(l))
	for i, s := range l {
		sl[i] = s.(string)
	}
	return sl
}

type buildResult struct {
	imageID  string
	metadata string
}

func createBuiltWithOptions(dockerCli command.Cli, in buildOptions) (res *buildResult, err error) {
	ctx := appcontext.Context()

	ctx, end, err := tracing.TraceCurrentCommand(ctx, "build")
	if err != nil {
		return nil, err
	}
	defer func() {
		end(err)
	}()

	noCache := false
	if in.noCache != nil {
		noCache = *in.noCache
	}
	pull := false
	if in.pull != nil {
		pull = *in.pull
	}

	opts := build.Options{
		Inputs: build.Inputs{
			ContextPath:    in.contextPath,
			DockerfilePath: in.dockerfileName,
			InStream:       os.Stdin,
		},
		BuildArgs:   in.buildArgs,
		ExtraHosts:  in.extraHosts,
		ImageIDFile: in.imageIDFile,
		Labels:      in.labels,
		NetworkMode: in.networkMode,
		NoCache:     noCache,
		Pull:        pull,
		ShmSize:     in.shmSize,
		Tags:        in.tags,
		Target:      in.target,
		Ulimits:     in.ulimits,
	}

	platforms, err := platformutil.Parse(in.platforms)
	if err != nil {
		return nil, err
	}
	opts.Platforms = platforms

	opts.Session = append(opts.Session, authprovider.NewDockerAuthProvider(os.Stderr))

	secrets, err := buildflags.ParseSecretSpecs(in.secrets)
	if err != nil {
		return nil, err
	}
	opts.Session = append(opts.Session, secrets)

	sshSpecs := in.ssh
	if len(sshSpecs) == 0 && buildflags.IsGitSSH(in.contextPath) {
		sshSpecs = []string{"default"}
	}
	ssh, err := buildflags.ParseSSHSpecs(sshSpecs)
	if err != nil {
		return nil, err
	}
	opts.Session = append(opts.Session, ssh)

	outputs := in.outputs
	opts.Exports, err = exportentry.TypedEntries(outputs).ToBuildkit()
	if err != nil {
		return nil, err
	}

	cacheImports, err := enrichCacheEntry(in.cacheFrom)
	if err != nil {
		return nil, err
	}
	opts.CacheFrom = cacheImports

	cacheExports, err := enrichCacheEntry(in.cacheTo)
	if err != nil {
		return nil, err
	}
	opts.CacheTo = cacheExports

	allow, err := buildflags.ParseEntitlements(in.allow)
	if err != nil {
		return nil, err
	}
	opts.Allow = allow

	// key string used for kubernetes "sticky" mode
	contextPathHash, err := filepath.Abs(in.contextPath)
	if err != nil {
		contextPathHash = in.contextPath
	}

	mf, err := os.CreateTemp("", "metadatafile")
	if err != nil {
		return nil, err
	}
	defer os.Remove(mf.Name())
	cf := consolefile.WithPrefix(ctx, os.Stderr, tflog.Info)
	imageID, err := commands.BuildTargets(ctx, dockerCli, map[string]build.Options{commands.DefaultTargetName: opts}, in.progress, contextPathHash, in.builder, in.metadataFile, cf)
	if err != nil {
		return nil, err
	}

	metadata, err := ioutil.ReadFile(mf.Name())
	if err != nil {
		return nil, err
	}

	return &buildResult{
		imageID:  imageID,
		metadata: string(metadata),
	}, nil
}

func transportName(output exportentry.TypedEntry) (string, error) {
	if output.Type == "" {
		panic("Output type missing")
	}
	var transportName string
	switch output.Type {
	case client.ExporterDocker:
		transportName = "docker"
	case client.ExporterOCI:
		transportName = "oci-archive"
	case client.ExporterTar:
		transportName = "tarball"
	case client.ExporterLocal:
		transportName = "dir"
	case client.ExporterImage:
		transportName = "docker-daemon"
	default:
		return "", fmt.Errorf("unknown type: %s", output.Type)
	}
	return transportName, nil
}

func deleteBuiltImage(ctx context.Context, dockerCli command.Cli, output exportentry.TypedEntry, iid string) error {

	if output.Type == "" {
		panic("Output type no set")
	}
	var reference string
	switch output.Type {
	case client.ExporterDocker:
		fallthrough
	case client.ExporterOCI:
		os.Remove(output.Dest.Value)
		return nil
	case client.ExporterImage:
		return nil
	case client.ExporterTar:
		reference = output.Dest.Value
	case client.ExporterLocal:
		dir, err := ioutil.ReadDir(output.Dest.Value)
		if err != nil {
			return fmt.Errorf("deleting %s failed: %w", output.Dest.Value, err)
		}
		for _, d := range dir {
			os.RemoveAll(path.Join([]string{output.Dest.Value, d.Name()}...))
		}
		return nil
	default:
		return fmt.Errorf("unknown type: %s", output.Type)
	}

	transportName, err := transportName(output)
	if err != nil {
		return err
	}
	transport := transports.Get(transportName)
	if transport == nil {
		return errors.Errorf(`Invalid output, unknown transport "%s". Known transports are: %v`, transportName, transports.ListNames())
	}
	ref, err := transport.ParseReference(reference)
	if err != nil {
		return err
	}

	registryHostname := "FOO"

	sys, err := newSystemContext(dockerCli, registryHostname)
	if err != nil {
		return err
	}

	return retry.RetryIfNecessary(ctx, func() error {
		return ref.DeleteImage(ctx, sys)
	}, &retry.RetryOptions{
		MaxRetry: 2,
	})
}

func enrichCacheEntry(in []client.CacheOptionsEntry) ([]client.CacheOptionsEntry, error) {
	imports := make([]client.CacheOptionsEntry, 0, len(in))
	for _, e := range in {

		if !addGithubToken(&e) {
			continue
		}
		imports = append(imports, e)
	}
	return imports, nil
}

func addGithubToken(ci *client.CacheOptionsEntry) bool {
	if ci.Type != "gha" {
		return true
	}
	if _, ok := ci.Attrs["token"]; !ok {
		if v, ok := os.LookupEnv("ACTIONS_RUNTIME_TOKEN"); ok {
			ci.Attrs["token"] = v
		}
	}
	if _, ok := ci.Attrs["url"]; !ok {
		if v, ok := os.LookupEnv("ACTIONS_CACHE_URL"); ok {
			ci.Attrs["url"] = v
		}
	}
	return ci.Attrs["token"] != "" && ci.Attrs["url"] != ""
}

func newSystemContext(dockerCli command.Cli, registryHostname string) (*imagetypes.SystemContext, error) {
	ctx := &imagetypes.SystemContext{}
	ac, err := dockerCli.ConfigFile().GetAuthConfig(registryHostname)
	if err != nil {
		return nil, err
	}
	/*
		ctx := opts.global.newSystemContext()
			ctx.DockerCertPath = opts.dockerCertPath
			ctx.OCISharedBlobDirPath = opts.sharedBlobDir
			ctx.AuthFilePath = opts.shared.authFilePath
			ctx.DockerDaemonHost = opts.dockerDaemonHost
			ctx.DockerDaemonCertPath = opts.dockerCertPath
			if opts.dockerImageOptions.authFilePath.Present() {
				ctx.AuthFilePath = opts.dockerImageOptions.authFilePath.Value()
			}
			if opts.deprecatedTLSVerify != nil && opts.deprecatedTLSVerify.tlsVerify.Present() {
				// If both this deprecated option and a non-deprecated option is present, we use the latter value.
				ctx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!opts.deprecatedTLSVerify.tlsVerify.Value())
			}
			if opts.tlsVerify.Present() {
				ctx.DockerDaemonInsecureSkipTLSVerify = !opts.tlsVerify.Value()
			}
			if opts.tlsVerify.Present() {
				ctx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(!opts.tlsVerify.Value())
			}
			if opts.credsOption.Present() && opts.noCreds {
				return nil, errors.New("creds and no-creds cannot be specified at the same time")
			}
			if opts.userName.Present() && opts.noCreds {
				return nil, errors.New("username and no-creds cannot be specified at the same time")
			}
			if opts.credsOption.Present() && ac.Username != "" {
				return nil, errors.New("creds and username cannot be specified at the same time")
			}
	*/
	// if any of username or password is present, then both are expected to be present
	if ac.Username != "" && ac.Password != "" {
		if ac.Username != "" {
			return nil, errors.New("password must be specified when username is specified")
		}
		return nil, errors.New("username must be specified when password is specified")
	}
	/*
		if opts.credsOption.Present() {
			var err error
			ctx.DockerAuthConfig, err = getDockerAuth(opts.credsOption.Value())
			if err != nil {
				return nil, err
			}
		} else if opts.userName.Present() {
			ctx.DockerAuthConfig = &types.DockerAuthConfig{
				Username: opts.userName.Value(),
				Password: opts.password.Value(),
			}
		}
		if opts.registryToken.Present() {
			ctx.DockerBearerRegistryToken = opts.registryToken.Value()
		}
		if opts.noCreds {
			ctx.DockerAuthConfig = &types.DockerAuthConfig{}
		}
	*/
	return ctx, nil
}
