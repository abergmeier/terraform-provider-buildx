package resources

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/moby/buildkit/client"
)

var (
	validOutputTypesMap = map[string]interface{}{
		client.ExporterDocker: nil,
		client.ExporterImage:  nil,
		client.ExporterLocal:  nil,
		client.ExporterOCI:    nil,
		client.ExporterTar:    nil,
		//"registry":            nil,
	}
	validOutputTypeList = []string{}

	validEntitlementsMap = map[string]interface{}{
		"security.insecure": nil,
		"network.host":      nil,
	}
	validEntitlementsList = []string{}
	dest                  = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
	}
)

func init() {
	for k := range validOutputTypesMap {
		validOutputTypeList = append(validOutputTypeList, k)
	}
	sort.Strings(validOutputTypeList)

	for k := range validEntitlementsMap {
		validEntitlementsList = append(validEntitlementsList, k)
	}
	sort.Strings(validEntitlementsList)
}

func outputDockerResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"dest": dest,
			"context": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func outputImageResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Image name",
			},
			"unpack": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "unpack image after creation (for use with containerd)",
			},
			"compression": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "choose compression type for layers newly created and cached, gzip is default value. estargz should be used with `oci-mediatypes=true.`",
				Default:     "gzip",
			},
			"compression_level": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "compression level for gzip, estargz (0-9) and zstd (0-22)",
			},
			"force_compression": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "forcefully apply `compression` option to all layers (including already existing layers)",
			},
			"buildinfo": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "choose [build dependency](https://github.com/moby/buildkit/blob/master/docs/build-repro.md#build-dependencies) version to export",
				Default:     "all",
			},
			"use_oci_mediatypes": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "use OCI mediatypes in configuration JSON instead of Docker's",
			},
		},
	}
}

func outputLocalResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"dest": dest,
		},
	}
}

func outputOciResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"dest": dest,
		},
	}
}

func outputTarResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"dest": dest,
		},
	}
}

func outputEntry() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			client.ExporterDocker: {
				Type:     schema.TypeSet,
				Elem:     outputDockerResource(),
				Optional: true,
				ForceNew: true,
				ExactlyOneOf: []string{
					"output.0." + client.ExporterDocker,
					"output.0." + client.ExporterImage,
					"output.0." + client.ExporterLocal,
					"output.0." + client.ExporterOCI,
					"output.0." + client.ExporterTar,
				},
			},
			client.ExporterImage: {
				Type:     schema.TypeSet,
				Elem:     outputImageResource(),
				Optional: true,
				ForceNew: true,
			},
			client.ExporterLocal: {
				Type:     schema.TypeSet,
				Elem:     outputLocalResource(),
				Optional: true,
				ForceNew: true,
			},
			client.ExporterOCI: {
				Type:     schema.TypeSet,
				Elem:     outputOciResource(),
				Optional: true,
				ForceNew: true,
			},
			client.ExporterTar: {
				Type:     schema.TypeSet,
				Elem:     outputTarResource(),
				Optional: true,
				ForceNew: true,
			},
		},
		Description: "For entries of `type` `gha`, GitHub Action credentials are automatically added to attrs.",
	}
}

func cacheEntryResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:     schema.TypeString,
				Elem:     schema.TypeString,
				Optional: true,
			},
			"attrs": {
				Type:     schema.TypeMap,
				Elem:     schema.TypeString,
				Required: true,
			},
		},
		Description: "For entries of `type` `gha`, GitHub Action credentials are automatically added to attrs.",
	}
}

func cacheResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"from": {
				Type:        schema.TypeList,
				Elem:        cacheEntryResource(),
				Optional:    true,
				Description: "External cache sources",
			},
			"to": {
				Type:        schema.TypeList,
				Elem:        cacheEntryResource(),
				Optional:    true,
				Description: "Cache export destinations",
			},
		},
	}
}

func BuiltResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"instance": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: `Builder instance name`,
			},
			"context": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"file": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the Dockerfile (default: `PATH/Dockerfile`). See https://docs.docker.com/engine/reference/commandline/build/#specify-a-dockerfile--f",
			},
			"output": {
				Type:        schema.TypeList,
				Elem:        outputEntry(),
				MaxItems:    1,
				MinItems:    1,
				Required:    true,
				Description: "Output destination",
				ForceNew:    true,
			},
			"allow": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				ForceNew:    true,
				Description: fmt.Sprintf("Allow extra privileged entitlement (`%s`)", strings.Join(validEntitlementsList, "`, `")),
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				ForceNew:    true,
				Description: "Name and optionally a tag (format: `name:tag`). See https://docs.docker.com/engine/reference/commandline/build/#tag-an-image--t",
			},
			"build_args": {
				Type:        schema.TypeMap,
				Elem:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Set build-time variables. See https://docs.docker.com/engine/reference/commandline/build/#set-build-time-variables---build-arg",
			},
			"labels": {
				Type:        schema.TypeMap,
				Elem:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Metadata for an image",
			},
			"cache": {
				Type:        schema.TypeSet,
				Elem:        cacheResource(),
				Optional:    true,
				MaxItems:    1,
				Description: "Cache settings",
				ForceNew:    true,
			},
			"iid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Image ID",
			},
			"metadata": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Build result metadata",
			},
		},
		Description:   "Starts a build using BuildKit. This resource is similar to the `docker build` command and takes similar arguments.",
		CreateContext: createBuilt,
		ReadContext:   readBuilt,
		DeleteContext: deleteBuilt,
	}
}
