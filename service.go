package autonats

import (
	"go/ast"
	"strings"
)

type service struct {
	InterfaceID string
	Name        string
	Methods     []*serviceMethod
	Imports     map[string]string
	Basedir     string
	PackageName string
	FileName    string
}

func (s *service) fromInterface(iface *ast.InterfaceType, file *ast.File) {
	s.Methods = make([]*serviceMethod, iface.Methods.NumFields())
	s.Imports = make(map[string]string)
	s.FileName = file.Name.Name

	for i, m := range iface.Methods.List {
		method := new(serviceMethod)
		method.fromFuncType(m.Type.(*ast.FuncType))
		method.Name = m.Names[0].Name
		s.Methods[i] = method
	}

	s.combineImports(file)
}

func (s *service) combineImports(file *ast.File) {
	imports := make(map[string]bool)

	for _, m := range s.Methods {
		for k := range m.imports {
			imports[k] = true
		}
	}

	for k := range imports {
		var name, path string

		for _, i := range file.Imports {
			if i.Name != nil {
				if i.Name.Name == k {
					name = i.Name.Name
					path = i.Path.Value
					break
				}
			} else {
				valSplit := strings.Split(strings.ReplaceAll(i.Path.Value, "\"", ""), "/")
				if valSplit[len(valSplit)-1] == k {
					path = i.Path.Value
					break
				}
			}
		}

		if path == "" {
			panic("empty path!")
		}

		path = strings.ReplaceAll(path, "\"", "")

		s.Imports[path] = name
	}
}
