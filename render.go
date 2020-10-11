package autonats

import (
	"bytes"
	"errors"
	"fmt"
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
	JsonLib                     string
	Tracing                     bool
}

func Render(data *RenderData) error {
	if data == nil || len(data.Services) == 0 {
		return errors.New("no data found to render")
	}

	data.Imports = append(data.Imports,
		"github.com/zyra/autonats",
		"github.com/nats-io/nats.go",
		"time",
		"github.com/json-iterator/go",
	)

	if data.Tracing {
		data.Imports = append(data.Imports,
			"github.com/nats-io/not.go",
			"github.com/opentracing/opentracing-go",
			"github.com/opentracing/opentracing-go/ext",
			"github.com/opentracing/opentracing-go/log")
	}

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
		return fmt.Errorf("failed to execute template: %s", err.Error())
	}

	out, err := format.Source(buff.Bytes())

	if err != nil {
		_ = ioutil.WriteFile(outFile, buff.Bytes(), 0655)
		return fmt.Errorf("failed to run gofmt on generated source: %s", err.Error())
	}

	fmt.Printf("rendering data to %s\n", outFile)

	if err := ioutil.WriteFile(outFile, out, 0655); err != nil {
		return fmt.Errorf("failed to write file: %s", err.Error())
	} else {
		return nil
	}
}
