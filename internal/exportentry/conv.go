package exportentry

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/moby/buildkit/client"
)

var (
	Extractors = map[string]func(interface{}) ([]Entry, error){
		client.ExporterDocker: toDockerExportEntry,
		client.ExporterImage:  toImageExportEntry,
		client.ExporterLocal:  toLocalExportEntry,
		client.ExporterOCI:    toOCIExportEntry,
		client.ExporterTar:    toTarExportEntry,
	}
)

func toDockerExportEntry(i interface{}) ([]Entry, error) {
	set := i.(*schema.Set)
	res := make([]Entry, set.Len())
	for i, di := range set.List() {
		m := di.(map[string]interface{})
		res[i].Type = client.ExporterDocker
		toExportEntryStringValue(m, "dest", &res[i].Dest)
		toExportEntryStringValue(m, "context", &res[i].Context)
	}

	return res, nil
}

func toExportEntryBoolValue(m map[string]interface{}, key string, v *bool) {
	f, ok := m[key]
	if !ok {
		return
	}
	*v = f.(bool)
}

func toImageExportEntry(i interface{}) ([]Entry, error) {
	set := i.(*schema.Set)
	res := make([]Entry, set.Len())
	for i, di := range set.List() {
		m := di.(map[string]interface{})
		res[i].Type = client.ExporterImage
		toExportEntryStringValue(m, "name", &res[i].Name)
		toExportEntryBoolValue(m, "unpack", &res[i].Unpack)
		toExportEntryStringValue(m, "compression", &res[i].Compression)
		toExportEntryIntValue(m, "compression_level", &res[i].CompressionLevel)
		toExportEntryBoolValue(m, "force_compression", &res[i].ForceCompression)
		toExportEntryBoolValue(m, "use_oci_mediatypes", &res[i].OCIMediatypes)
		toExportEntryStringValue(m, "buildinfo", &res[i].Buildinfo)
	}

	return res, nil
}

func toLocalExportEntry(i interface{}) ([]Entry, error) {
	set := i.(*schema.Set)
	res := make([]Entry, set.Len())
	for i, di := range set.List() {
		m := di.(map[string]interface{})
		res[i].Type = client.ExporterLocal
		toExportEntryStringValue(m, "dest", &res[i].Dest)
	}

	return res, nil
}

func toOCIExportEntry(i interface{}) ([]Entry, error) {
	set := i.(*schema.Set)
	res := make([]Entry, set.Len())
	for i, di := range set.List() {
		m := di.(map[string]interface{})
		res[i].Type = client.ExporterOCI
		toExportEntryStringValue(m, "dest", &res[i].Dest)
	}

	return res, nil
}

func toTarExportEntry(i interface{}) ([]Entry, error) {
	set := i.(*schema.Set)
	res := make([]Entry, set.Len())
	for i, di := range set.List() {
		m := di.(map[string]interface{})
		res[i].Type = client.ExporterTar
		toExportEntryStringValue(m, "dest", &res[i].Dest)
	}

	return res, nil
}

func toExportEntryIntValue(m map[string]interface{}, key string, v *int) {
	f, ok := m[key]
	if !ok {
		return
	}
	*v = f.(int)
}

func toExportEntryStringValue(m map[string]interface{}, key string, v *string) {
	f, ok := m[key]
	if !ok {
		return
	}
	*v = f.(string)
}
