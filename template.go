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
	"subject": func(srv *Service, method *Method) string {
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
	"combine": func(strs ...string) string {
		return strings.Join(strs, "")
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

{{- define "server_interface" }}
    type {{ .Name }}Server interface {
    {{- range $index, $method := .Methods }}
        {{ $method.Name }}({{ template "params" $method }}) {{ template "results" $method }}
    {{- end }}
    }
{{ end -}}

{{ define "type_ref" -}}
    {{ if .Array }}[]{{ if .Pointer }}*{{ end }}{{ end }}
    {{- if .TypePackage }}{{ .TypePackage }}.{{ end }}{{ .Type }}
{{- end }}

{{ define "type_ref_full" -}}
	{{ if and (not .Array) .Pointer }}*{{ end }}{{ template "type_ref" . }}
{{- end }}

package {{ .PackageName }}

import (
{{ range .Imports }}	"{{ . }}"
{{ end -}}
)

const timeout = time.Second * {{ .Timeout }}

{{ range $srv := .Services }}
    {{ template "server_interface" $srv }}

    {{- $handlerName := (printf "%sHandler" (lower $srv.Name)) }}
    {{- $serverName := (printf "%sServer" $srv.Name) }}
    {{- $clientName := (printf "%sClient" $srv.Name) }}

    type {{ $handlerName }} struct {
        Server {{ $serverName }}
        nc *nats.Conn
        runners []*autonats.Runner
    }

    func (h *{{ $handlerName }}) Run(ctx context.Context) error {
        h.runners = make([]*autonats.Runner, {{ len $srv.Methods }}, {{ len $srv.Methods }})
		tracer := opentracing.GlobalTracer()

        {{- range $index, $method := $srv.Methods }}
            {{- $subject := subject $srv $method }}
            if runner, err := autonats.StartRunner(ctx, h.nc, "{{ $subject }}", "autonats", {{ $method.HandlerConcurrency }}, func(msg *nats.Msg) {
                t := not.NewTraceMsg(msg)
				sc, err := tracer.Extract(opentracing.Binary, t)
				if err != nil {
					return
				}
		
				replySpan := tracer.StartSpan("Autonats {{ $serverName }}", ext.SpanKindRPCServer, ext.RPCServerOption(sc))
				ext.MessageBusDestination.Set(replySpan, msg.Subject)
				defer replySpan.Finish()
				innerCtx, _ := context.WithTimeout(ctx, timeout)
				innerCtxT := opentracing.ContextWithSpan(innerCtx, replySpan)


				{{ $hasResult := gt (len $method.Results) 1 }}
				
				{{ if $hasResult }}
				var result {{ template "type_ref_full" (index $method.Results 0) }}
				{{ end }}

				{{ $hasParam := gt (len $method.Params) 1 }}
				{{ if $hasParam }}

				{{ $param := index $method.Params 1 }}

				{{ if eq $param.Type "string" -}}
				{{ if $hasResult }}result, {{ end }} err = h.Server.{{ $method.Name }}(innerCtxT, string(msg.Data))
				{{ else }}
                var data {{ template "type_ref" $param }}
                if err = {{ $.JsonLib }}.Unmarshal(msg.Data, &data); err != nil {
					replySpan.LogFields(log.Error(err))
                    return
                }
				{{ if $hasResult }}result, {{ end }} err = h.Server.{{ $method.Name }}(innerCtxT, {{ if and $param.Pointer (not $param.Array) }}&{{ end }}data)
				{{ end }}

				{{ else }}
				{{ if $hasResult }}result, {{ end }} err = h.Server.{{ $method.Name }}(innerCtxT)
				{{ end }}
				
				reply := autonats.GetReply()
				defer autonats.PutReply(reply)

				{{ $result := index $method.Results 0 }}
				{{ $nilResult := nilResult $result }}
	
				if err != nil {
					replySpan.LogFields(log.Event("Handler returned error"))
					reply.Error = []byte(err.Error())
				{{ if $hasResult }}
				} else if result != {{ $nilResult }} {
					replySpan.LogFields(log.Event("Handler returned a result"))
					if err := reply.MarshalAndSetData(result); err != nil {
						replySpan.LogFields(log.Error(err))
						return
					}
				{{ end }}
				}

				replyData, err := reply.MarshalBinary()

				if err != nil {
					replySpan.LogFields(log.Error(err))
					return
				}
		
				replySpan.LogFields(log.Event("Sending reply"))
				if err := msg.Respond(replyData); err != nil {
					replySpan.LogFields(log.Error(err))
					return
				}
            }); err != nil {
				{{- if gt $index 0 }}
				h.Shutdown()
				{{ end -}}
                return err
            } else {
                h.runners[{{ $index }}] = runner
            }
        {{ end }}

        return nil
    }

    func (h *{{ $handlerName }}) Shutdown() {
        for i := range h.runners {
			if h.runners[i] != nil {
            	_ = h.runners[i].Shutdown()
			}
        }
    }

    func New{{ $srv.Name }}Handler(server {{ $serverName }}, nc *nats.Conn) autonats.Handler {
        return &{{ $handlerName }}{
            Server: server,
            nc: nc,
        }
    }

    type {{ $clientName }} struct {
        nc *nats.Conn
        log autonats.Logger
    }

    {{ range $index, $method := .Methods }}
        func (client *{{ $clientName }}) {{ $method.Name }}({{ template "params" $method }}) {{ template "results" $method }} {
        {{- $subject := subject $srv $method }}
        {{- $hasResult := gt (len $method.Results) 1 }}
	
		{{ $nilResult := "" }}
	
		{{ if $hasResult }}
			{{ $result := index $method.Results 0 }}
			{{ $nilResult = combine (nilResult $result)  ", "}}
		{{ end }}
		
		reqSpan, reqCtx := opentracing.StartSpanFromContext(ctx, "Autonats {{ $clientName }} Request", ext.SpanKindRPCClient)
		ext.MessageBusDestination.Set(reqSpan, "{{ $subject }}")
		defer reqSpan.Finish()
		reqSpan.LogFields(log.Event("Starting request"))
	
		var t not.TraceMsg
		var err error
	
		if err = opentracing.GlobalTracer().Inject(reqSpan.Context(), opentracing.Binary, &t); err != nil {
			reqSpan.LogFields(log.Error(err))
			return {{ $nilResult }} err
		}


		{{ $hasParam := gt (len $method.Params) 1 }}
		{{ if $hasParam }}
			{{ $param := index $method.Params 1 }}
			{{ $isString := eq $param.Type "string" }}
			
			{{ if not $isString }}
				var data []byte
				data, err = jsoniter.Marshal({{ $param.Name }})
				if err != nil {
					reqSpan.LogFields(log.Error(err))
					return {{ $nilResult }} err
				}
			{{ end }}
	
			if _, err = t.Write(
			{{- if eq $param.Type "string" -}}
				[]byte({{ $param.Name }})
			{{- else -}}
				data
			{{- end -}}
			); err != nil {
				reqSpan.LogFields(log.Error(err))
				return {{ $nilResult }} err
			}
		{{ end }}	

		reqCtx, cancelFn := context.WithTimeout(reqCtx, timeout)
		defer cancelFn()

		var replyMsg *nats.Msg
		if replyMsg, err = client.nc.RequestWithContext(ctx, "{{ $subject }}", t.Bytes()); err != nil {
			reqSpan.LogFields(log.Error(err))
			return {{ $nilResult }} err
		}

		reqSpan.LogFields(log.Event("Reply received"))
		reply := autonats.GetReply()
		defer autonats.PutReply(reply)
		
		if err := reply.UnmarshalBinary(replyMsg.Data); err != nil {
			reqSpan.LogFields(log.Error(err))
			return {{ $nilResult }} err
		}

		if err := reply.GetError(); err != nil {
			return {{ $nilResult }} err
		}

		

		{{ if $hasResult }}
			{{ $result := (index $method.Results 0) }}
			
			{{ if eq $result.Type "string" }}
				return reply.GetDataAsString()
			{{ else }}
		
			var result {{ template "type_ref" $result }}
			if err := reply.UnmarshalData(&result); err != nil {
				return {{ $nilResult }} err
			}
	
			return {{ if and $result.Pointer (not $result.Array) }}&{{ end }}result, nil
			{{ end }}	

		{{ else }}
            return nil
        {{- end }}
        }
    {{ end }}

{{ end }}
`),
)
