package autonats

import (
	"go/ast"
	"strings"
)

type Param struct {
	Name, Type, TypePackage string
	Pointer, Array          bool
	RequiredImports         map[string]bool
}

func ParseParam(f *ast.Field) *Param {
	param := &Param{
		RequiredImports: make(map[string]bool),
	}

	switch p := f.Type.(type) {
	case *ast.SelectorExpr:
		param.typeFromSelectorExpr(p)

	case *ast.Ident:
		param.typeFromIdent(p)

	case *ast.StarExpr:
		param.typeFromStarExpr(p)

	case *ast.ArrayType:
		param.typeFromArray(p)

	default:
		panic("unhandled type")
	}

	if len(f.Names) > 0 {
		param.Name = f.Names[0].Name
	}

	return param
}

func (param *Param) typeFromSelectorExpr(sExp *ast.SelectorExpr) {
	if sExp.X != nil {
		ident := sExp.X.(*ast.Ident)
		param.TypePackage = ident.Name
		param.RequiredImports[ident.Name] = true
	}

	param.Type = sExp.Sel.Name

	if param.Name == "" {
		param.Name = strings.ToLower(sExp.Sel.Name)
	}
}

func (param *Param) typeFromStarExpr(sExp *ast.StarExpr) {
	param.Pointer = true

	switch sExp.X.(type) {
	case *ast.Ident:
		param.Type = sExp.X.(*ast.Ident).Name
		param.Name = strings.ToLower(param.Type)
	case *ast.SelectorExpr:
		sExpX := sExp.X.(*ast.SelectorExpr)
		param.typeFromSelectorExpr(sExpX)
		param.Name = strings.ToLower(sExpX.Sel.Name)

	default:
		panic("unhandled type")
	}
}

func (param *Param) typeFromIdent(ident *ast.Ident) {
	param.Type = ident.Name

	if param.Name == "" {
		param.Name = strings.ToLower(param.Name)
	}
}

func (param *Param) typeFromArray(arr *ast.ArrayType) {
	param.Array = true

	switch arr.Elt.(type) {
	case *ast.SelectorExpr:
		param.typeFromSelectorExpr(arr.Elt.(*ast.SelectorExpr))

	case *ast.Ident:
		param.typeFromIdent(arr.Elt.(*ast.Ident))

	case *ast.StarExpr:
		param.typeFromStarExpr(arr.Elt.(*ast.StarExpr))

	default:
		panic("unhandled type")
	}
}
