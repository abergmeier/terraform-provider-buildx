package resources

import (
	"bytes"
	"fmt"

	"github.com/docker/buildx/driver"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

var drivers bytes.Buffer

func init() {
	for _, d := range driver.GetFactories() {
		if len(drivers.String()) > 0 {
			drivers.WriteString(", ")
		}
		drivers.WriteString(fmt.Sprintf("`%s`", d.Name()))
	}
}

func buildkitResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"flags": {
				Type:        schema.TypeList,
				Elem:        schema.TypeString,
				Optional:    true,
				Description: `Flags for buildkitd daemon`,
			},
		},
	}
}

func driverResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: fmt.Sprintf("Driver to use (available: %s)", drivers.String()),
			},
			"opt": {
				Type:        schema.TypeMap,
				Elem:        schema.TypeString,
				Optional:    true,
				Description: `Options for the driver`,
			},
			"keep_state": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: `Keep BuildKit state on delete`,
			},
		},
	}
}

func InstanceResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Description:      `Builder instance name`,
				ValidateDiagFunc: validateName,
				ExactlyOneOf:     []string{"name", "generate_name"},
			},
			"generate_name": {
				Type:         schema.TypeBool,
				Optional:     true,
				ForceNew:     true,
				Description:  `Generate a Build instance name`,
				ExactlyOneOf: []string{"name", "generate_name"},
			},
			"context": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   `If neither a context nor an endpoint is specified the current Docker configuration is used for determining the context/endpoint value.`,
				ConflictsWith: []string{"endpoint"},
				ForceNew:      true,
			},
			"endpoint": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   `If neither a context nor an endpoint is specified the current Docker configuration is used for determining the context/endpoint value.`,
				ConflictsWith: []string{"context"},
				ForceNew:      true,
			},
			"buildkit": {
				Type:     schema.TypeSet,
				Elem:     buildkitResource(),
				Optional: true,
				MaxItems: 1,
				ForceNew: true,
			},
			"driver": {
				Type:     schema.TypeSet,
				Elem:     driverResource(),
				MinItems: 1,
				MaxItems: 1,
				ForceNew: true,
				Required: true,
			},
			"bootstrap": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: `Boot builder after creation`,
			},
			"generated_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: `Result of name generation`,
			},
		},
		Description: `Makes a new builder instance pointing to a docker context or endpoint, where context is the name of a context from ` + "`docker context ls`" + ` and endpoint is the address for docker socket (eg. ` + "`DOCKER_HOST`" + ` value).
By default, the current Docker configuration is used for determining the context/endpoint value.	
Builder instances are isolated environments where builds can be invoked.`,
		CreateContext: createInstance,
		ReadContext:   readInstance,
		DeleteContext: deleteInstance,
	}
}

func validateName(v interface{}, p cty.Path) diag.Diagnostics {

	if v.(string) == "default" {
		return diag.FromErr(errors.Errorf("default is a reserved name and cannot be used to identify builder instance"))
	}
	return nil
}
