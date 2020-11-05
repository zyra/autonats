package autonats

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
)

// Parser config
type ParserConfig struct {
	BaseDir            string // Directory containing interfaces to scan
	DefaultTimeout     int    // Timeout for NATS requests
	OutputFileName     string // Output file name
	DefaultConcurrency int    // Default handler concurrency
	Tracing            bool   // Generate tracing code
}

// Parser object
type Parser struct {
	config      *ParserConfig
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
func NewParser(config *ParserConfig) *Parser {
	return &Parser{
		config:      config,
		services:    make([]*Service, 0),
		rawPackages: make(map[string]*ast.Package),
		packages:    make(map[string]*Package),
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

		for _, m := range service.Methods {
			if m.Timeout <= 0 {
				m.Timeout = par.config.DefaultTimeout
			}

			if m.HandlerConcurrency <= 0 {
				m.HandlerConcurrency = par.config.DefaultConcurrency
			}
		}

		pkg.AddService(service)
	}

	par.services = services
	par.packages = packages
}

func (par *Parser) Render() error {
	imports := make([]string, 0)

	for pk := range par.packages {
		for k := range par.packages[pk].Imports {
			imports = append(imports, k)
		}
	}

	data := RenderData{
		FileName: par.config.OutputFileName,
		Path:     par.config.BaseDir,
		Services: par.services,
		Imports:  imports,
		Timeout:  par.config.DefaultTimeout,
		JsonLib:  "jsoniter",
		Tracing:  par.config.Tracing,
	}

	return Render(&data)
}
