package autonats

import (
	"go/ast"
	"strings"
)

type methodParam struct {
	Name            string
	Type            string
	TypePackage     string
	Pointer         bool
	Array           bool
	RequiredImports map[string]bool
}

func (p *methodParam) parse(f *ast.Field) {
	p.RequiredImports = make(map[string]bool)

	switch f.Type.(type) {
	case *ast.SelectorExpr:
		p.typeFromSelectorExpr(f.Type.(*ast.SelectorExpr))

	case *ast.Ident:
		p.typeFromIdent(f.Type.(*ast.Ident))

	case *ast.StarExpr:
		p.typeFromStarExpr(f.Type.(*ast.StarExpr))

	case *ast.ArrayType:
		p.typeFromArray(f.Type.(*ast.ArrayType))

	default:
		panic("unhandled type")
	}

	if len(f.Names) > 0 {
		p.Name = f.Names[0].Name
	}
}

func (p *methodParam) typeFromSelectorExpr(sExp *ast.SelectorExpr) {
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

func (p *methodParam) typeFromStarExpr(sExp *ast.StarExpr) {
	p.Pointer = true

	switch sExp.X.(type) {
	case *ast.Ident:
		p.Type = sExp.X.(*ast.Ident).Name
		p.Name = strings.ToLower(p.Type)
	case *ast.SelectorExpr:
		sExpX := sExp.X.(*ast.SelectorExpr)
		p.typeFromSelectorExpr(sExpX)
		p.Name = strings.ToLower(sExpX.Sel.Name)

	default:
		panic("unhandled type")
	}
}

func (p *methodParam) typeFromIdent(ident *ast.Ident) {
	p.Type = ident.Name

	if p.Name == "" {
		p.Name = strings.ToLower(p.Name)
	}
}

func (p *methodParam) typeFromArray(arr *ast.ArrayType) {
	p.Array = true

	switch arr.Elt.(type) {
	case *ast.SelectorExpr:
		p.typeFromSelectorExpr(arr.Elt.(*ast.SelectorExpr))

	case *ast.Ident:
		p.typeFromIdent(arr.Elt.(*ast.Ident))

	case *ast.StarExpr:
		p.typeFromStarExpr(arr.Elt.(*ast.StarExpr))

	default:
		panic("unhandled type")
	}
}
