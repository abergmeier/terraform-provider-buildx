package resources

import "github.com/hashicorp/terraform-plugin-framework/types"

type instanceBuildkitData struct {
	Flags []string `tfsdk:"flags"`
}

type instanceDriverData struct {
	Name      string            `tfsdk:"name"`
	Opt       map[string]string `tfsdk:"opt"`
	KeepState types.Bool        `tfsdk:"keep_state"`
}

type instanceResourceData struct {
	Bootstrap     types.Bool            `tfsdk:"bootstrap"`
	Context       types.String          `tfsdk:"context"`
	Id            types.String          `tfsdk:"id"`
	Name          types.String          `tfsdk:"name"`
	GenerateName  types.Bool            `tfsdk:"generate_name"`
	Endpoint      types.String          `tfsdk:"endpoint"`
	Buildkit      *instanceBuildkitData `tfsdk:"buildkit"`
	Driver        *instanceDriverData   `tfsdk:"driver"`
	GeneratedName types.String          `tfsdk:"generated_name"`
}
