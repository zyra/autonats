package autonats

import (
	"go/ast"
	"strings"
)

type MethodParam struct {
	Name            string
	Type            string
	TypePackage     string
	Pointer         bool
	Array           bool
	RequiredImports map[string]bool
}

func (p *MethodParam) Parse(f *ast.Field) {
	p.RequiredImports = make(map[string]bool)

	switch f.Type.(type) {
	case *ast.SelectorExpr:
		p.TypeFromSelectorExpr(f.Type.(*ast.SelectorExpr))

	case *ast.Ident:
		p.TypeFromIdent(f.Type.(*ast.Ident))

	case *ast.StarExpr:
		p.TypeFromStarExpr(f.Type.(*ast.StarExpr))

	case *ast.ArrayType:
		p.TypeFromArray(f.Type.(*ast.ArrayType))

	default:
		panic("unhandled type")
	}

	if len(f.Names) > 0 {
		p.Name = f.Names[0].Name
	}
}

func (p *MethodParam) TypeFromSelectorExpr(sExp *ast.SelectorExpr) {
	if sExp.X != nil {
		ident := sExp.X.(*ast.Ident)
		p.TypePackage = ident.Name
		p.RequiredImports[ident.Name] = true
	}

	p.Type = sExp.Sel.Name

	if p.Name == "" {
		p.Name = strings.ToLower(sExp.Sel.Name)
	}
}

func (p *MethodParam) TypeFromStarExpr(sExp *ast.StarExpr) {
	p.Pointer = true

	switch sExp.X.(type) {
	case *ast.Ident:
		p.Type = sExp.X.(*ast.Ident).Name
		p.Name = strings.ToLower(p.Type)
	case *ast.SelectorExpr:
		sExpX := sExp.X.(*ast.SelectorExpr)
		p.TypeFromSelectorExpr(sExpX)
		p.Name = strings.ToLower(sExpX.Sel.Name)

	default:
		panic("unhandled type")
	}
}

func (p *MethodParam) TypeFromIdent(ident *ast.Ident) {
	p.Type = ident.Name

	if p.Name == "" {
		p.Name = strings.ToLower(p.Name)
	}
}

func (p *MethodParam) TypeFromArray(arr *ast.ArrayType) {
	p.Array = true

	switch arr.Elt.(type) {
	case *ast.SelectorExpr:
		p.TypeFromSelectorExpr(arr.Elt.(*ast.SelectorExpr))

	case *ast.Ident:
		p.TypeFromIdent(arr.Elt.(*ast.Ident))

	case *ast.StarExpr:
		p.TypeFromStarExpr(arr.Elt.(*ast.StarExpr))

	default:
		panic("unhandled type")
	}
}
