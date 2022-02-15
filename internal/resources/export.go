package resources

import (
	"fmt"
	"strconv"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
)

type ExportEntry struct {
	Type             string
	Name             string
	Dest             string
	OCIMediatypes    bool
	Unpack           bool
	Compression      string
	CompressionLevel int
	ForceCompression bool
	Buildinfo        string
}

type ExportEntries []ExportEntry

func (ex *ExportEntry) ToBuildkit() (out client.ExportEntry, err error) {
	exports := []string{
		fmt.Sprintf("type=%s,dest=%s", ex.Type, ex.Dest),
	}
	parsed, err := build.ParseOutput(exports)
	out.Output = parsed[0].Output
	out.OutputDir = parsed[0].OutputDir
	out.Type = ex.Type
	attrs := make(map[string]string, 8)
	setExportEntryStringValue(attrs, ex.Name, "name")
	attrs["push"] = "false"
	setExportEntryBoolValue(attrs, ex.OCIMediatypes, "oci-mediatypes")
	setExportEntryBoolValue(attrs, ex.Unpack, "unpack")
	setExportEntryStringValue(attrs, ex.Compression, "compression")
	setExportEntryIntValue(attrs, ex.CompressionLevel, "compression-level")
	setExportEntryBoolValue(attrs, ex.ForceCompression, "force-compression")
	setExportEntryStringValue(attrs, ex.Buildinfo, "buildinfo")
	out.Attrs = attrs
	return
}

func (ex *ExportEntries) ToBuildkit() (out []client.ExportEntry, err error) {
	for _, e := range *ex {
		ee, err := e.ToBuildkit()
		if err != nil {
			return nil, err
		}
		out = append(out, ee)
	}
	return out, nil
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
