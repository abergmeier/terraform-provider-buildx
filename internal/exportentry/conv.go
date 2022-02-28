package exportentry

import (
	"github.com/moby/buildkit/client"
)

var (
	Extractors = map[string]func(interface{}) ([]Entry, error){
		client.ExporterDocker: func(i interface{}) ([]Entry, error) {
			return toExportEntries(i, toDockerExportEntry)
		},
		client.ExporterImage: func(i interface{}) ([]Entry, error) {
			return toExportEntries(i, toImageExportEntry)
		},
		client.ExporterLocal: func(i interface{}) ([]Entry, error) {
			return toExportEntries(i, toLocalExportEntry)
		},
		client.ExporterOCI: func(i interface{}) ([]Entry, error) {
			return toExportEntries(i, toOCIExportEntry)
		},
		client.ExporterTar: func(i interface{}) ([]Entry, error) {
			return toExportEntries(i, toTarExportEntry)
		},
	}
)

func toExportEntries(i interface{}, f func(map[string]interface{}) Entry) ([]Entry, error) {
	l := i.([]interface{})
	res := make([]Entry, len(l))
	for i, di := range l {
		res[i] = f(di.(map[string]interface{}))
	}
	return res, nil
}

func toDockerExportEntry(m map[string]interface{}) (res Entry) {
	res.Type = client.ExporterDocker
	toExportEntryStringValue(m, "dest", &res.Dest)
	toExportEntryStringValue(m, "context", &res.Context)
	return
}

func toExportEntryBoolValue(m map[string]interface{}, key string, v *bool) {
	f, ok := m[key]
	if !ok {
		return
	}
	*v = f.(bool)
}

func toImageExportEntry(m map[string]interface{}) (res Entry) {
	res.Type = client.ExporterImage
	toExportEntryStringValue(m, "name", &res.Name)
	toExportEntryBoolValue(m, "unpack", &res.Unpack)
	toExportEntryStringValue(m, "compression", &res.Compression)
	toExportEntryIntValue(m, "compression_level", &res.CompressionLevel)
	toExportEntryBoolValue(m, "force_compression", &res.ForceCompression)
	toExportEntryBoolValue(m, "use_oci_mediatypes", &res.OCIMediatypes)
	toExportEntryStringValue(m, "buildinfo", &res.Buildinfo)
	return
}

func toLocalExportEntry(m map[string]interface{}) (res Entry) {
	res.Type = client.ExporterLocal
	toExportEntryStringValue(m, "dest", &res.Dest)
	return
}

func toOCIExportEntry(m map[string]interface{}) (res Entry) {
	res.Type = client.ExporterOCI
	toExportEntryStringValue(m, "dest", &res.Dest)
	return
}

func toTarExportEntry(m map[string]interface{}) (res Entry) {
	res.Type = client.ExporterTar
	toExportEntryStringValue(m, "dest", &res.Dest)
	return
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
