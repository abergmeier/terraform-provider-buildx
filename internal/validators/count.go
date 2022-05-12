package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func HasAtMostOneEntry(attributeName string) tfsdk.AttributeValidator {
	return &hasAtMostOneEntry{
		attributeName: attributeName,
	}
}

func HasExactlyOneEntry(attributeName string) tfsdk.AttributeValidator {
	return &hasExactlyOneEntry{
		attributeName: attributeName,
	}
}

type hasAtMostOneEntry struct {
	attributeName string
}

func (v *hasAtMostOneEntry) Description(context.Context) string {
	return fmt.Sprintf("Ensures that %s has no or one entry", v.attributeName)
}

func (v *hasAtMostOneEntry) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *hasAtMostOneEntry) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	value, err := req.AttributeConfig.ToTerraformValue(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot convert config to Terraform value", err.Error())
		return
	}

	var m map[string]tftypes.Value
	err = value.As(&m)
	if err == nil {

		if len(m) <= 1 {
			return
		}

		resp.Diagnostics.AddAttributeError(req.AttributePath, "Has more than one element", "")
		return

	}

	var l []tftypes.Value
	err = value.As(&l)
	if err == nil {

		if len(l) <= 1 {
			return
		}

		resp.Diagnostics.AddAttributeError(req.AttributePath, "Has more than one element", "")
		return
	}

	resp.Diagnostics.AddAttributeError(req.AttributePath, "Value not a known collection", "")
}

type hasExactlyOneEntry struct {
	attributeName string
}

func (v *hasExactlyOneEntry) Description(context.Context) string {
	return fmt.Sprintf("Ensures that %s has exactly one entry", v.attributeName)
}

func (v *hasExactlyOneEntry) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *hasExactlyOneEntry) Validate(ctx context.Context, req tfsdk.ValidateAttributeRequest, resp *tfsdk.ValidateAttributeResponse) {
	value, err := req.AttributeConfig.ToTerraformValue(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Cannot convert config to Terraform value", err.Error())
		return
	}

	var m map[string]tftypes.Value
	err = value.As(&m)
	if err == nil {

		if len(m) == 1 {
			return
		}

		resp.Diagnostics.AddAttributeError(req.AttributePath, "Has not exactly one element", "")
		return
	}

	var l []tftypes.Value
	err = value.As(&l)
	if err == nil {

		if len(l) == 1 {
			return
		}

		resp.Diagnostics.AddAttributeError(req.AttributePath, "Has not exactly one element", "")
		return
	}

	resp.Diagnostics.AddAttributeError(req.AttributePath, "Value not a known collection", "")
}
