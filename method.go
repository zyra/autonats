package autonats

import "go/ast"

type ServiceMethod struct {
	Name    string
	Params  []*MethodParam
	Results []*MethodParam
	Reply   bool
	imports map[string]bool
}

func (m *ServiceMethod) FromFuncType(fx *ast.FuncType) {
	nParams := fx.Params.NumFields()
	nResults := fx.Results.NumFields()

	m.Params = make([]*MethodParam, nParams, nParams)
	m.Results = make([]*MethodParam, nResults, nResults)
	m.imports = make(map[string]bool)

	for ii, p := range fx.Params.List {
		param := new(MethodParam)
		param.Parse(p)
		m.Params[ii] = param

		for k, _ := range param.RequiredImports {
			m.imports[k] = true
		}
	}

	if fx.Results != nil && len(fx.Results.List) > 0 {
		m.Reply = true
		for ii, r := range fx.Results.List {
			result := new(MethodParam)
			result.Parse(r)
			m.Results[ii] = result

			for k, _ := range result.RequiredImports {
				m.imports[k] = true
			}
		}
	}
}