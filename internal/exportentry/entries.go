package exportentry

import "github.com/moby/buildkit/client"

type Entries []Entry

func (ex *Entries) ToBuildkit() (out []client.ExportEntry, err error) {
	for _, e := range *ex {
		ee, err := e.ToBuildkit()
		if err != nil {
			return nil, err
		}
		out = append(out, ee)
	}
	return out, nil
}
