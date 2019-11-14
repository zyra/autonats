package autonats

import "go/ast"

type serviceMethod struct {
	Name    string
	Params  []*methodParam
	Results []*methodParam
	Reply   bool
	imports map[string]bool
}

func (m *serviceMethod) fromFuncType(fx *ast.FuncType) {
	nParams := fx.Params.NumFields()
	nResults := fx.Results.NumFields()

	m.Params = make([]*methodParam, nParams, nParams)
	m.Results = make([]*methodParam, nResults, nResults)
	m.imports = make(map[string]bool)

	for ii, p := range fx.Params.List {
		param := new(methodParam)
		param.parse(p)
		m.Params[ii] = param

		for k := range param.RequiredImports {
			m.imports[k] = true
		}
	}

	if fx.Results != nil && len(fx.Results.List) > 0 {
		m.Reply = true
		for ii, r := range fx.Results.List {
			result := new(methodParam)
			result.parse(r)
			m.Results[ii] = result

			for k := range result.RequiredImports {
				m.imports[k] = true
			}
		}
	}
}
