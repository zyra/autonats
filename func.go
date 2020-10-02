package autonats

import (
	"go/ast"
)

type Function struct {
	Name               string
	Params             []*Param
	Results            []*Param
	imports            map[string]bool
	HandlerConcurrency int
}

func FuncFromType(fxField *ast.Field) *Function {
	fx := fxField.Type.(*ast.FuncType)

	nParams := fx.Params.NumFields()
	nResults := fx.Results.NumFields()

	m := &Function{
		Params:             make([]*Param, nParams, nParams),
		Results:            make([]*Param, nResults, nResults),
		imports:            make(map[string]bool),
		Name:               fxField.Names[0].Name,
		HandlerConcurrency: 5,
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
