package exportentry

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
)

type Entry struct {
	Type             string
	Name             string
	Context          string
	Dest             string
	OCIMediatypes    bool
	Unpack           bool
	Compression      string
	CompressionLevel int
	ForceCompression bool
	Buildinfo        string
}

func (ex *Entry) ToBuildkit() (out client.ExportEntry, err error) {
	if ex.Type == "" {
		err = errors.New("unspecified type not supported")
		return
	}
	exports := []string{
		fmt.Sprintf("type=%s,dest=%s", ex.Type, ex.Dest),
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
