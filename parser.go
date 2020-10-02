package autonats

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

// Parser config
type ParserConfig struct {
	BaseDir        string // Directory containing interfaces to scan
	DefaultTimeout int    // Timeout for NATS requests
	OutputFileName string // Output file name
}

// Parser object
type Parser struct {
	services    []*Service
	rawPackages map[string]*ast.Package
	packages    map[string]*Package
}

func ParseDir(path string) (map[string]*ast.Package, error) {
	fileSet := token.NewFileSet()

	if !filepath.IsAbs(path) {
		if baseDir, err := filepath.Abs(path); err != nil {
			return nil, err
		} else {
			path = baseDir
		}
	}

	return parser.ParseDir(fileSet, path, nil, 4)
}

// Creates a new parser with the provided config
func NewParser() *Parser {
	return &Parser{
		services:    make([]*Service, 0),
		packages:    make(map[string]*Package),
		rawPackages: make(map[string]*ast.Package),
	}
}

func (par *Parser) ParseDir(path string) error {
	packages, err := ParseDir(path)

	if err != nil {
		return err
	}

	par.AddRawPackages(packages)

	return nil
}

func (par *Parser) AddRawPackages(packages map[string]*ast.Package) {
	for k, v := range packages {
		par.rawPackages[k] = v
	}
}

// Runs the parser and outputs generated code to file
func (par *Parser) Run() {
	packages := make(map[string]*Package)
	services := make([]*Service, 0)

	for _, v := range par.rawPackages {
		services = append(services, ServicesFromPkg(v)...)
	}

	for _, service := range services {
		pkg, ok := packages[service.FileName]

		if !ok {
			pkg = PackageFromService(service)
			packages[service.FileName] = pkg
		}

		pkg.AddService(service)
	}

	par.services = services
	par.packages = packages
}

func (par *Parser) Render(baseDir, outfile string, timeout int) {
	imports := make([]string, 0)

	for i := range par.packages {
		for k, _ := range par.packages[i].Imports {
			imports = append(imports, k)
		}
	}

	data := RenderData{
		FileName:    outfile,
		Path:        baseDir,
		Services:    par.services,
		Imports:     imports,
		Timeout:     timeout,
	}

	Render(&data)
}
