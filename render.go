package autonats

import (
	"bytes"
	"go/format"
	"io/ioutil"
	"path/filepath"
	"sort"
)

type RenderData struct {
	PackageName, FileName, Path string
	Services                    []*Service
	Imports                     []string
	Timeout                     int
}

func Render(data *RenderData) error {
	data.Imports = append(data.Imports, "github.com/zyra/autonats", "github.com/nats-io/nats.go", "json", "time")

	sort.Strings(data.Imports)
	sort.Slice(data.Services, func(i, j int) bool {
		return data.Services[i].Name < data.Services[j].Name
	})

	data.PackageName = data.Services[0].PackageName

	data.Timeout = 5

	outFile := filepath.Join(data.Path, data.FileName)

	b := make([]byte, 0)
	buff := bytes.NewBuffer(b)

	err := tmplService.Execute(buff, data)

	if err != nil {
		return err
	}

	out, err := format.Source(buff.Bytes())

	if err != nil {
		return err
	}

	return ioutil.WriteFile(outFile, out, 0655)
}
