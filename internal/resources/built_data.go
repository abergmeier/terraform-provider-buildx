package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type cacheEntryData struct {
	Type  types.String      `tfsdk:"type"`
	Attrs map[string]string `tfsdk:"attrs"`
}

type builtOutputData struct {
	Docker *struct {
		Context types.String `tfsdk:"context"`
		Dest    types.String `tfsdk:"dest"`
	} `tfsdk:"docker"`
	Image *struct {
		Name types.String `tfsdk:"name"`
	} `tfsdk:"image"`
	Local *struct {
		Dest types.String `tfsdk:"dest"`
	} `tfsdk:"local"`
	OCI *struct {
		Dest types.String `tfsdk:"dest"`
	} `tfsdk:"oci"`
	Tar *struct {
		Dest types.String `tfsdk:"dest"`
	} `tfsdk:"tar"`
}

type builtResourceData struct {
	Allow     types.Set         `tfsdk:"allow"`
	BuildArgs map[string]string `tfsdk:"build_args"`
	Cache     *struct {
		From []cacheEntryData `tfsdk:"from"`
		To   []cacheEntryData `tfsdk:"to"`
	} `tfsdk:"cache"`
	Context  types.String      `tfsdk:"context"`
	File     types.String      `tfsdk:"file"`
	Id       types.String      `tfsdk:"id"`
	Iid      types.String      `tfsdk:"iid"`
	Instance types.String      `tfsdk:"instance"`
	Labels   map[string]string `tfsdk:"labels"`
	Metadata types.String      `tfsdk:"metadata"`
	Tags     []string          `tfsdk:"tags"`
	Output   *builtOutputData  `tfsdk:"output"`
}
