package resources

import (
	"bytes"
	"context"
	"fmt"

	"github.com/abergmeier/terraform-provider-buildx/internal/meta"
	"github.com/abergmeier/terraform-provider-buildx/internal/validators"
	"github.com/docker/buildx/driver"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	commaSeparatedDriverNames bytes.Buffer
	buildkitAttributes        = map[string]tfsdk.Attribute{
		"flags": {
			Type: types.ListType{
				ElemType: types.StringType,
			},
			Optional:    true,
			Description: `Flags for buildkitd daemon`,
		},
	}
	InstanceType = &instanceType{}
)

func init() {
	for _, d := range driver.GetFactories() {
		if len(commaSeparatedDriverNames.String()) > 0 {
			commaSeparatedDriverNames.WriteString(", ")
		}
		commaSeparatedDriverNames.WriteString(fmt.Sprintf("`%s`", d.Name()))
	}
}

type instanceType struct{}

func driverAttributes(commaSeparatedDriverNames string) map[string]tfsdk.Attribute {
	return map[string]tfsdk.Attribute{
		"name": {
			Type:        types.StringType,
			Required:    true,
			Description: fmt.Sprintf("Driver to use (available: %s)", commaSeparatedDriverNames),
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
		"opt": {
			Type: types.MapType{
				ElemType: types.StringType,
			},
			Optional:    true,
			Description: `Options for the driver`,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
		"keep_state": {
			Type:        types.BoolType,
			Optional:    true,
			Description: `Keep BuildKit state on delete`,
			PlanModifiers: tfsdk.AttributePlanModifiers{
				tfsdk.RequiresReplace(),
			},
		},
	}
}

func instanceSchema(commaSeparatedDriverNames string) tfsdk.Schema {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Buildx Instance",
		Blocks:              map[string]tfsdk.Block{},
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				Type:     types.StringType,
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				MarkdownDescription: `Builder instance name`,
				Validators: []tfsdk.AttributeValidator{
					&validators.ValidateInstanceName{},
				},
			},
			"generate_name": {
				Type:     types.BoolType,
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				MarkdownDescription: `Generate a Build instance name`,
			},
			"buildkit": {
				Optional:   true,
				Attributes: tfsdk.SingleNestedAttributes(buildkitAttributes),
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"context": {
				Type:                types.StringType,
				Optional:            true,
				MarkdownDescription: `If neither a context nor an endpoint is specified the current Docker configuration is used for determining the context/endpoint value.`,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"driver": {
				Attributes: tfsdk.SingleNestedAttributes(driverAttributes(commaSeparatedDriverNames)),
				Optional:   true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"endpoint": {
				Type:                types.StringType,
				Optional:            true,
				MarkdownDescription: `If neither a context nor an endpoint is specified the current Docker configuration is used for determining the context/endpoint value.`,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"bootstrap": {
				Type:     types.BoolType,
				Optional: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				MarkdownDescription: `Boot builder after creation`,
			},
			"generated_name": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"id": {
				Type:     types.StringType,
				Computed: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
		},
		Description: `Makes a new builder instance pointing to a docker context or endpoint, where context is the name of a context from ` + "`docker context ls`" + ` and endpoint is the address for docker socket (eg. ` + "`DOCKER_HOST`" + ` value).
By default, the current Docker configuration is used for determining the context/endpoint value.	
Builder instances are isolated environments where builds can be invoked.`,
	}
}

func (r *instanceType) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return instanceSchema(commaSeparatedDriverNames.String()), nil
}

func (r *instanceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return &instanceResource{
		m: p.(meta.HasData).Data(),
	}, nil
}
