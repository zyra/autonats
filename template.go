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
	Server *{{ $serverName }}
	nc *nats.Conn
	runners []*autonats.Runner
}

func (h *{{ $handlerName }}) Run(ctx context.Context) error {
	h.runners = make([]*autonats.Runner, {{ len $srv.Methods }}, {{ len $srv.Methods }})
{{- range $index, $method := $srv.Methods }}
{{- $subject := subject $srv $method }}
	if runner, err := autonats.StartRunner(ctx, h.nc, {{ $subject }}, "autonats", {{ $method.HandlerConcurrency }}, func(msg *nats.Msg) (interface{}, error) {
{{- $param := index $method.Params 1 }}
		var data {{ if $param.Array }}[]{{ if $param.Pointer }}*{{ end }}{{ end }}
{{- if $param.TypePackage }}{{ $param.TypePackage }}.{{ end }}{{ $param.Type }}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		} else {
			innerCtx, _ := context.WithTimeout(time.Second * {{ $.Timeout }})
			return h.Server.{{ $method.Name }}(innerCtx, {{ if and $param.Pointer (not $param.Array) }}&{{ end }}data)
		}
	}); err != nil {
		return err;
	} else {
		h.runners[{{ $index }}] = runner
	}
{{ end }}	
}

func (h *{{ $handlerName }}) Shutdown() {
	for i := h.runners {
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
	nc *nats.Conn
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

	if err := SendRequest(ctx, client.nc, "{{ $subject }}", {{ (index $method.Params 1).Name }}, &data); err != nil {
		return nil, err
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
