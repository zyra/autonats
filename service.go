package autonats

import (
	"fmt"
	"go/ast"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DocPrefix = "@nats:"

type Service struct {
	InterfaceID string
	Name        string
	Methods     []*Function
	Imports     map[string]string
	Basedir     string
	PackageName string
	FileName    string
}

func (svc *Service) combineImports(with []*ast.ImportSpec) {
	imports := make(map[string]bool)

	for _, m := range svc.Methods {
		for k := range m.imports {
			imports[k] = true
		}
	}

	for k := range imports {
		var name, path string

		for _, i := range with {
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

		svc.Imports[path] = name
	}
}

type ServiceConfig struct {
	Name    string
	Timeout time.Duration
}

func ServiceConfigFromDoc(doc *ast.CommentGroup) ServiceConfig {
	text := doc.Text()

	rgx := regexp.MustCompile(fmt.Sprintf(`(?im)%s([a-z0-9-_]+)\s([a-z0-9-_]+)`, DocPrefix))
	matches := rgx.FindAllSubmatch([]byte(text), -1)

	args := make(map[string][]string)

	for _, match := range matches {
		if len(match) < 3 {
			// line is too short to be ours
			continue
		}

		ar := make([]string, len(match)-2)
		key := string(match[1])
		args[key] = ar

		for ai, a := range match[2:] {
			ar[ai] = string(a)
		}
	}

	return ServiceConfig{
		Name:    args["server"][0],
		Timeout: time.Second * 3,
	}
}

func ServicesFromFile(pkgName, fileName string, file *ast.File) []*Service {
	services := make([]*Service, 0)

	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		decl, ok, val := findServiceDecl(node)

		if !ok {
			return val
		}

		typeSpec, ok := decl.Specs[0].(*ast.TypeSpec)

		if !ok {
			log.Println("invalid spec type")
			return false
		}

		iface, ok := findServiceIface(typeSpec)

		if !ok {
			return false
		}

		svcConfig := ServiceConfigFromDoc(decl.Doc)

		methods := make([]*Function, iface.Methods.NumFields())

		for i, m := range iface.Methods.List {
			methods[i] = FuncFromType(m)
		}

		service := Service{
			InterfaceID: typeSpec.Name.Name,
			Name:        svcConfig.Name,
			Methods:     methods,
			Imports:     make(map[string]string),
			Basedir:     filepath.Dir(fileName),
			FileName:    file.Name.Name,
			PackageName: pkgName,
		}

		service.combineImports(file.Imports)

		services = append(services, &service)

		return true
	})

	return services
}

func findServiceDecl(node ast.Node) (decl *ast.GenDecl, ok, value bool) {
	decl, ok = node.(*ast.GenDecl)

	if !ok {
		return nil, false, true
	}

	if decl.Doc == nil || !strings.Contains(decl.Doc.Text(), DocPrefix) {
		return nil, false, false
	}

	if len(decl.Specs) != 1 {
		log.Println("invalid number of specs")
		return nil, false, false
	}

	return decl, true, false
}

func findServiceIface(typeSpec *ast.TypeSpec) (*ast.InterfaceType, bool) {
	iface, ok := typeSpec.Type.(*ast.InterfaceType)

	if !ok {
		log.Println("couldn't find an interface")
		return nil, false
	}

	if iface.Methods.NumFields() == 0 {
		log.Println("interface has no methods")
		return nil, false
	}

	return iface, true
}

func ServicesFromPkg(v *ast.Package) []*Service {
	services := make([]*Service, 0)

	for fk, fv := range v.Files {
		services = append(services, ServicesFromFile(v.Name, fk, fv)...)
	}

	return services
}
