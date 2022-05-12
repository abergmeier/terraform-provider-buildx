package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/abergmeier/terraform-provider-buildx/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	validEntitlementsList                    = []string{}
	BuiltType             tfsdk.ResourceType = &builtResourceType{}
	outputAttributes                         = map[string]tfsdk.Attribute{
		client.ExporterDocker: {
			Attributes: tfsdk.SingleNestedAttributes(outputDockerAttributes),
			Optional:   true,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
		client.ExporterImage: {
			Attributes: tfsdk.SingleNestedAttributes(outputImageAttributes),
			Optional:   true,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
		client.ExporterLocal: {
			Attributes: tfsdk.SingleNestedAttributes(outputLocalAttributes),
			Optional:   true,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
		client.ExporterOCI: {
			Attributes: tfsdk.SingleNestedAttributes(outputOciAttributes),
			Optional:   true,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
		client.ExporterTar: {
			Attributes: tfsdk.SingleNestedAttributes(outputTarAttributes),
			Optional:   true,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
	}
	outputDockerAttributes = map[string]tfsdk.Attribute{
		"dest": {
			Type:        types.StringType,
			Optional:    true,
			Description: `destination path where tarball will be written. If not specified the tar will be loaded automatically to the current docker instance.`,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
		"context": {
			Type:        types.StringType,
			Optional:    true,
			Description: `name for the docker context where to import the result`,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
	}
	outputImageAttributes = map[string]tfsdk.Attribute{
		"name": {
			Type:        types.StringType,
			Required:    true,
			Description: "Image name",
		},
	}
	/*
			"unpack": {
				Type:        types.BoolType,
				Optional:    true,
				Description: "unpack image after creation (for use with containerd)",
			},
			"compression": {
				Type:                types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "choose compression type for layers newly created and cached, gzip is default value. estargz should be used with `oci-mediatypes=true.`",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					modifier.StringDefault("gzip"),
				},
			},
			"compression_level": {
				Type:        types.NumberType,
				Optional:    true,
				Description: "compression level for gzip, estargz (0-9) and zstd (0-22)",
			},
			"force_compression": {
				Type:                types.BoolType,
				Optional:            true,
				MarkdownDescription: "forcefully apply `compression` option to all layers (including already existing layers)",
			},
			"buildinfo": {
				Type:                types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "choose [build dependency](https://github.com/moby/buildkit/blob/master/docs/build-repro.md#build-dependencies) version to export",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					modifier.StringDefault("all"),
				},
			},
			"use_oci_mediatypes": {
				Type:        types.BoolType,
				Optional:    true,
				Description: "use OCI mediatypes in configuration JSON instead of Docker's",
			},
		}
	*/
	outputLocalAttributes = map[string]tfsdk.Attribute{
		"dest": {
			Type:        types.StringType,
			Optional:    true,
			Description: `destination directory where files will be written`,
		},
	}
	outputOciAttributes = map[string]tfsdk.Attribute{
		"dest": {
			Type:        types.StringType,
			Optional:    true,
			Description: `destination path where tarball will be written`,
		},
	}
	outputTarAttributes = map[string]tfsdk.Attribute{
		"dest": {
			Type:        types.StringType,
			Optional:    true,
			Description: `destination path where tarball will be written`,
		},
	}
	cacheAttributes = map[string]tfsdk.Attribute{
		"from": {
			Attributes:  tfsdk.ListNestedAttributes(cacheEntryAttributes, tfsdk.ListNestedAttributesOptions{}),
			Required:    true,
			Description: "Use an external cache source for a build.",
		},
		"to": {
			Attributes:  tfsdk.ListNestedAttributes(cacheEntryAttributes, tfsdk.ListNestedAttributesOptions{}),
			Required:    true,
			Description: "Export build cache to an external cache destination.",
		},
	}
	cacheEntryAttributes = map[string]tfsdk.Attribute{
		"type": {
			Type:                types.StringType,
			Optional:            true,
			MarkdownDescription: "Supported types are `registry`, `local` and `gha`.",
		},
		"attrs": {
			Type: types.MapType{
				ElemType: types.StringType,
			},
			Optional: true,
		},
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

type builtResourceType struct{}

func (t builtResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		Attributes: map[string]tfsdk.Attribute{
			"instance": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				Description: `Builder instance name`,
			},
			"context": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"output": {
				Attributes:          tfsdk.SingleNestedAttributes(outputAttributes),
				Required:            true,
				Description:         "Output destination. For entries of type gha, GitHub Action credentials are automatically added to attrs.",
				MarkdownDescription: "Output destination. For entries of `type` `gha`, GitHub Action credentials are automatically added to attrs.",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				Validators: []tfsdk.AttributeValidator{
					&validators.HasOneOutputType{},
				},
			},
			"cache": {
				Attributes:  tfsdk.SingleNestedAttributes(cacheAttributes),
				Optional:    true,
				Description: "Cache settings",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"file": {
				Type:     types.StringType,
				Required: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				MarkdownDescription: "Name of the Dockerfile (default: `PATH/Dockerfile`). See https://docs.docker.com/engine/reference/commandline/build/#specify-a-dockerfile--f",
			},
			"allow": {
				Type: types.SetType{
					ElemType: types.StringType,
				},
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				MarkdownDescription: fmt.Sprintf("Allow extra privileged entitlement (`%s`)", strings.Join(validEntitlementsList, "`, `")),
			},
			"tags": {
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				MarkdownDescription: "Name and optionally a tag (format: `name:tag`). See https://docs.docker.com/engine/reference/commandline/build/#tag-an-image--t",
			},
			"build_args": {
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				MarkdownDescription: "Set build-time variables. See https://docs.docker.com/engine/reference/commandline/build/#set-build-time-variables---build-arg",
			},
			"labels": {
				Type: types.MapType{
					ElemType: types.StringType,
				},
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				Description: "Metadata for an image",
			},
			"id": {
				Type:     types.StringType,
				Computed: true,
			},
			"iid": {
				Type:        types.StringType,
				Computed:    true,
				Description: "Image ID",
			},
			"metadata": {
				Type:        types.StringType,
				Computed:    true,
				Description: "Build result metadata",
			},
		},
		Description:         "Starts a build using BuildKit. This resource is similar to the docker build command and takes similar arguments.",
		MarkdownDescription: "Starts a build using BuildKit. This resource is similar to the `docker build` command and takes similar arguments.",
	}, nil
}

func (t builtResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return &builtResource{
		m: p.(meta.HasData).Data(),
	}, nil
}
