package exportentry

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
)

type TypedEntry struct {
	Entry
	Type string
}

type Entry struct {
	Name             types.String `tfsdk:"name"`
	Context          types.String `tfsdk:"context"`
	Dest             types.String `tfsdk:"dest"`
	UseOCIMediatypes types.Bool   `tfsdk:"use_oci_mediatypes"`
	Unpack           types.Bool   `tfsdk:"unpack"`
	Compression      types.String `tfsdk:"compression"`
	CompressionLevel types.Int64  `tfsdk:"compression_level"`
	ForceCompression types.Bool   `tfsdk:"force_compression"`
	Buildinfo        types.String `tfsdk:"buildinfo"`
}

func (ex *TypedEntry) ToBuildkit() (out client.ExportEntry, err error) {
	if ex.Type == "" {
		err = errors.New("unspecified type not supported")
		return
	}
	exports := []string{
		fmt.Sprintf("type=%s,dest=%s", ex.Type, ex.Dest.Value),
	}
	parsed, err := build.ParseOutput(exports)
	if err != nil {
		err = fmt.Errorf("parsing output (%s) failed: %w", exports[0], err)
		return
	}
	out.Output = parsed[0].Output
	out.OutputDir = parsed[0].OutputDir
	out.Type = ex.Type
	attrs := make(map[string]string, 8)
	setExportEntryStringValue(attrs, ex.Name.Value, "name")
	attrs["push"] = "false"
	setExportEntryBoolValue(attrs, ex.UseOCIMediatypes.Value, "oci-mediatypes")
	setExportEntryBoolValue(attrs, ex.Unpack.Value, "unpack")
	setExportEntryStringValue(attrs, ex.Compression.Value, "compression")
	setExportEntryIntValue(attrs, int(ex.CompressionLevel.Value), "compression-level")
	setExportEntryBoolValue(attrs, ex.ForceCompression.Value, "force-compression")
	setExportEntryStringValue(attrs, ex.Buildinfo.Value, "buildinfo")
	out.Attrs = attrs
	return
}

func setExportEntryBoolValue(attrs map[string]string, v bool, attrName string) {
	if v {
		attrs[attrName] = "true"
	} else {
		attrs[attrName] = "false"
	}
}

func setExportEntryIntValue(attrs map[string]string, v int, attrName string) {
	attrs[attrName] = strconv.Itoa(v)
}

func setExportEntryStringValue(attrs map[string]string, v, attrName string) {
	if v == "" {
		return
	}
	attrs[attrName] = v
}
