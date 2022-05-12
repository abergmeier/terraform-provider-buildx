package validators

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

type HasOneOutputType struct {
}

func (v *HasOneOutputType) Description(context.Context) string {
	return "Validate Output Type"
}

func (v *HasOneOutputType) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *HasOneOutputType) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {

	_, err := req.AttributeConfig.ToTerraformValue(ctx)
	if err != nil {
		panic(err)
	}

	/*
		var s string
		if in.IsNull() {
			return
		}
		err = in.As(&s)
		if err != nil {
			resp.Diagnostics.AddAttributeError(req.AttributePath, "Converting to string failed", err.Error())
			return
		}

		if s == "default" {
			resp.Diagnostics.AddAttributeError(req.AttributePath, "Invalid Instance name", "default is a reserved name and cannot be used to identify builder instance")
			return
		}
	*/
}
