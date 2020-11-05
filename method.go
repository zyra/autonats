package autonats

import (
	"go/ast"
)

// Describes a service method that's exposed to the service mesh
type Method struct {
	Name               string
	Params             []*Param
	Results            []*Param
	imports            map[string]bool
	HandlerConcurrency int // Method handler concurrency
	Timeout            int // Method timeout
}

func MethodFromField(field *ast.Field) *Method {
	fx := field.Type.(*ast.FuncType)

	nParams := fx.Params.NumFields()
	nResults := fx.Results.NumFields()

	m := &Method{
		Params:             make([]*Param, nParams, nParams),
		Results:            make([]*Param, nResults, nResults),
		imports:            make(map[string]bool),
		Name:               field.Names[0].Name,
		HandlerConcurrency: 0, // TODO: add custom tag/comment to define concurrency for each method
		Timeout:            0, // TODO: add custom  tag/comment to define timeout for each method
	}

	for ii, p := range fx.Params.List {
		par := ParseParam(p)
		m.Params[ii] = par

		for k := range par.RequiredImports {
			m.imports[k] = true
		}
	}

	if fx.Results != nil && len(fx.Results.List) > 0 {
		for ii, r := range fx.Results.List {
			result := ParseParam(r)
			m.Results[ii] = result

			for k := range result.RequiredImports {
				m.imports[k] = true
			}
		}
	}

	return m
}
