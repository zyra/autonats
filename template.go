package autonats

import (
	"fmt"
	"reflect"
	"strings"
	"text/template"
)

func isLastItem(array interface{}, index int) bool {
	return index == reflect.ValueOf(array).Len()-1
}

var funMap = template.FuncMap{
	"last":  isLastItem,
	"lower": strings.ToLower,
	"subject": func(srv *Service, method *Function) string {
		return fmt.Sprintf("autonats.%s.%s", srv.Name, method.Name)
	},
	"returnPointer": func(result *Param) bool {
		return !result.Array && result.Pointer
	},
	"nilResult": func(result *Param) string {
		if result.Array {
			return `nil`
		}

		switch result.Type {
		case "int", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float", "float64", "byte":
			return "0"
		case "bool":
			return "false"
		case "string":
			return `""`
		}

		return "nil"
	},
}

var tmplService = template.Must(
	template.New("outfile").Funcs(funMap).Parse(`
{{- define "params" }}
{{- $method := . }}
{{- range $pi, $p := $method.Params -}}
{{ $p.Name }} {{ if $p.Array }}[]{{ end }}{{ if $p.Pointer }}*{{ end }}{{ if $p.TypePackage }}{{ $p.TypePackage }}.{{ end }}{{ $p.Type }}
{{- if not (last $method.Params $pi) -}}, {{ end -}}
{{- end }}
{{- end -}}

{{- define "results" }}
{{- $method := . }}
{{- $multi := gt (len $method.Results) 1 }}
{{- if $multi }}({{ end }}
{{- range $pi, $p := $method.Results -}}
{{ if $p.Array }}[]{{ end }}{{ if $p.Pointer }}*{{ end }}{{ if $p.TypePackage }}{{ $p.TypePackage }}.{{ end }}{{ $p.Type }}
{{- if not (last $method.Results $pi) -}}, {{ end -}}
{{- end }}
{{- if $multi }}){{ end }}
{{- end -}}
package {{ .PackageName }}

import (
{{ range .Imports }}	"{{ . }}"
{{ end -}}
)

const timeout = time.Second * {{ .Timeout }}

{{ range $srv := .Services }}
type {{ $srv.Name }}Server interface {
{{- range $index, $method := .Methods }}
	{{ $method.Name }}({{ template "params" $method }}) {{ template "results" $method }}
{{- end }}
}

{{- $handlerName := (printf "%sHandler" (lower $srv.Name)) }}
{{- $serverName := (printf "%sServer" $srv.Name) }}

type {{ $handlerName }} struct {
	Server {{ $serverName }}
	nc *nats.Conn
	runners []*autonats.Runner
}

func (h *{{ $handlerName }}) Run(ctx context.Context) error {
	h.runners = make([]*autonats.Runner, {{ len $srv.Methods }}, {{ len $srv.Methods }})
{{- range $index, $method := $srv.Methods }}
{{- $subject := subject $srv $method }}
	if runner, err := autonats.StartRunner(ctx, h.nc, "{{ $subject }}", "autonats", {{ $method.HandlerConcurrency }}, func(msg *nats.Msg) (interface{}, error) {
{{- $param := index $method.Params 1 }}
		var data {{ if $param.Array }}[]{{ if $param.Pointer }}*{{ end }}{{ end }}
{{- if $param.TypePackage }}{{ $param.TypePackage }}.{{ end }}{{ $param.Type }}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		} else {
			innerCtx, _ := context.WithTimeout(ctx, time.Second * {{ $.Timeout }})
			return {{ if eq 1 (len $method.Results) }}nil, {{ end }}h.Server.{{ $method.Name }}(innerCtx, {{ if and $param.Pointer (not $param.Array) }}&{{ end }}data)
		}
	}); err != nil {
		return err;
	} else {
		h.runners[{{ $index }}] = runner
	}
{{ end }}

	return nil
}

func (h *{{ $handlerName }}) Shutdown() {
	for i := range h.runners {
		h.runners[i].Shutdown()
	}
}

func New{{ $srv.Name }}Handler(server {{ $serverName }}, nc *nats.Conn) autonats.Handler {
	return &{{ $handlerName }}{
		Server: server,
		nc: nc,
	}
}

type {{ $srv.Name }}Client struct {
	nc *nats.EncodedConn
	log autonats.Logger
}

{{ range $index, $method := .Methods }}
func (client *{{ $srv.Name }}Client) {{ $method.Name }}({{ template "params" $method }}) {{ template "results" $method }} {
{{- $subject := subject $srv $method }}
{{- $hasResult := gt (len $method.Results) 1 }}
{{- if $hasResult }}
{{- $result := (index $method.Results 0) }}
	var dest {{ if $result.Array }}[]{{ if $result.Pointer }}*{{ end }}{{ end }}
{{- if $result.TypePackage }}{{ $result.TypePackage }}.{{ end }}{{ $result.Type }}

	if err := autonats.SendRequest(ctx, client.nc, "{{ $subject }}", {{ (index $method.Params 1).Name }}, &dest); err != nil {
		return {{ nilResult (index $method.Results 0) }}, err
	} else {
		return {{ if and $result.Pointer (not $result.Array) }}&{{end}}dest, nil
	}
{{- else }}
	return autonats.SendRequest(ctx, client.nc, "{{ $subject }}", {{ (index $method.Params 1).Name }}, nil)
{{- end }}
}
{{ end }}

{{ end }}
`),
)
