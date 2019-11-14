package autonats

import (
	"bytes"
	. "github.com/dave/jennifer/jen"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

type ParserConfig struct {
	BaseDir     string
	NatsTimeout int
	OutFileName string
}

type Parser struct {
	*ParserConfig
	services []*Service
	rawPkgs  map[string]*ast.Package
	pkgs     map[string]*Package
}

type Package struct {
	Services       []*Service
	Imports        map[string]string
	Name           string
	BaseDir        string
	OriginFileName string
	FileName       string
}

func NewParser(config *ParserConfig) (p *Parser, err error) {
	p = &Parser{
		ParserConfig: config,
		services:     make([]*Service, 0),
		pkgs:         make(map[string]*Package),
	}

	fset := token.NewFileSet()

	if !filepath.IsAbs(config.BaseDir) {
		if config.BaseDir, err = filepath.Abs(config.BaseDir); err != nil {
			return nil, err
		}
	}

	if config.OutFileName == "" {
		config.OutFileName = "nats_client.go"
	} else if filepath.Ext(config.OutFileName) != ".go" {
		config.OutFileName += ".go"
	}

	if config.NatsTimeout == 0 {
		config.NatsTimeout = 3
	}

	if p.rawPkgs, err = parser.ParseDir(fset, config.BaseDir, nil, 4); err != nil {
		return nil, err
	} else {
		return p, nil
	}
}

func (p *Parser) Run() {
	p.loadServices()
	p.createPkgs()
	p.render()
}

func (p *Parser) loadServices() {
	for _, v := range p.rawPkgs {
		for fk, fv := range v.Files {
			ast.Inspect(fv, func(node ast.Node) bool {
				if node == nil {
					return false
				}

				decl, ok := node.(*ast.GenDecl)

				if ! ok {
					return true
				}

				if decl.Doc == nil || !strings.Contains(decl.Doc.Text(), "@nats:") {
					return false
				}

				if len(decl.Specs) != 1 {
					log.Println("invalid number of specs")
					return false
				}

				typeSpec, ok := decl.Specs[0].(*ast.TypeSpec)

				if ! ok {
					log.Println("invalid spec type")
					return false
				}

				iface, ok := typeSpec.Type.(*ast.InterfaceType)

				if ! ok {
					log.Println("couldn't find an interface")
					return false
				}

				if iface.Methods.NumFields() == 0 {
					log.Println("interface has no methods")
					return false
				}

				service := new(Service)
				service.PackageName = v.Name
				service.Basedir = filepath.Dir(fk)
				service.InterfaceID = typeSpec.Name.Name

				service.FromInterface(iface, fv)

				docText := decl.Doc.Text()

				rgx := regexp.MustCompile(`(?im)@nats:([a-z0-9-_]+)\s([a-z0-9-_]+)`)
				matches := rgx.FindAllSubmatch([]byte(docText), -1)

				args := make(map[string][]string)

				for _, arg := range matches {
					if len(arg) < 3 {
						continue
					}

					ar := make([]string, len(arg)-2)
					args[string(arg[1])] = ar

					for ai, a := range arg[2:] {
						ar[ai] = string(a)
					}
				}

				service.Name = args["server"][0]

				p.services = append(p.services, service)

				return true
			})
		}
	}
}

func (p *Parser) createPkgs() {
	for _, s := range p.services {
		_, ok := p.pkgs[s.FileName]

		if ! ok {
			p.pkgs[s.FileName] = &Package{
				Services:       make([]*Service, 0),
				Imports:        make(map[string]string),
				Name:           s.PackageName,
				BaseDir:        s.Basedir,
				OriginFileName: s.FileName,
				FileName:       s.FileName,
			}
		}

		p.pkgs[s.FileName].Services = append(p.pkgs[s.FileName].Services, s)

		for ik, iv := range s.Imports {
			p.pkgs[s.FileName].Imports[ik] = iv
		}
	}
}

func (p *Parser) render() {
	for _, pkg := range p.pkgs {
		t := NewFile(pkg.Name)

		t.PackageComment("// Code generated by autonats. DO NOT EDIT.")

		imps := make(map[string]string)

		for ik, iv := range pkg.Imports {
			if iv != "" {
				imps[iv] = ik
				t.ImportAlias(ik, iv)
			} else {
				split := strings.Split(ik, "/")
				iv = split[len(split)-1]
				imps[iv] = ik
				t.ImportName(ik, iv)
			}
		}

		t.ImportName("github.com/zyra/autonats", "autonats")

		for _, s := range pkg.Services {
			clientName := s.Name + "Client"
			handlerName := s.Name + "Handler"

			{
				t.Type().Id(clientName).Struct(
					Id("nc").Op("*").Qual("github.com/nats-io/nats.go", "Conn"),
					Id("l").Qual("github.com/zyra/autonats", "Logger"),
				)

				for _, m := range s.Methods {
					fx := t.Func().Params(Id("c").Op("*").Id(clientName))
					fx.Id(m.Name)

					paramCode := make([]Code, len(m.Params))

					for _, p := range m.Params {
						pc := Id(p.Name)

						if p.Array {
							pc.Op("[]")
						}

						if p.Pointer {
							pc.Op("*")
						}

						if p.TypePackage != "" {
							pc.Qual(imps[p.TypePackage], p.Type)
						} else {
							pc.Id(p.Type)
						}

						paramCode = append(paramCode, pc)
					}

					fx.Params(paramCode...)

					lenResults := len(m.Results)

					if lenResults > 0 {
						paramCode = make([]Code, lenResults)

						for _, p := range m.Results {
							pc := Add()

							if p.Type == "error" {
								pc.Id("e")
							} else {
								pc.Id("r")
							}

							if p.Array {
								pc.Op("[]")
							}

							if p.Pointer {
								pc.Op("*")
							}

							if p.TypePackage != "" {
								pc.Qual(imps[p.TypePackage], p.Type)
							} else {
								pc.Id(p.Type)
							}

							paramCode = append(paramCode, pc)
						}

						fx.Params(paramCode...)
					}

					fx.BlockFunc(func(g *Group) {
						g.Var().Id("d").Op("[]").Byte()

						if !m.Reply {
							g.Var().Id("e").Error()
						}

						if len(m.Params) > 0 {
							g.IfFunc(func(g *Group) {
								g.Id("d").Op(",").Id("e").Op("=").Qual(
									"encoding/json", "Marshal",
								).CallFunc(func(g *Group) {
									if !m.Params[0].Pointer || m.Params[0].Array {
										g.Add(Op("&").Id(m.Params[0].Name))
									} else {
										g.Id(m.Params[0].Name)
									}
								}).Op(";").Id("e").Op("!=").Nil().BlockFunc(func(g *Group) {
									if m.Reply {
										results := make([]Code, lenResults)

										for i := 0; i < lenResults-1; i++ {
											results[i] = Nil()
										}

										results[lenResults-1] = Qual("fmt", "Errorf").Call(
											Lit("unable to marshal request %s\n"), Id("e").Dot("Error").Call(),
										)

										g.Return(results...)
									} else {
										g.Id("c").Dot("l").Dot("Printf").Call(
											Lit("unable to marshal request %s\n"), Id("e").Dot("Error").Call(),
										)
									}
								})
							}).Else()
						}

						requestErrorBlock := func(g *Group) {
							if m.Reply {
								results := make([]Code, lenResults)

								for i := 0; i < lenResults-1; i++ {
									results[i] = Nil()
								}

								results[lenResults-1] = Id("e")

								g.Return(results...)
							} else {
								g.Id("c").Dot("l").Dot("Printf").Call(
									Lit("received error from request: %s\n"), Id("e").Dot("Error").Call(),
								)
							}
						}
						handleRequestParams := Op("&").Qual("github.com/zyra/autonats", "Request").Block(
							Id("Subject").Op(":").Lit("gonats." + s.Name + "." + m.Name).Op(",").Id("Data").Op(":").Id("d").Op(","),
						)

						if !m.Reply || len(m.Results) == 1 {
							g.If().Id("d").Op(",").Id("e").Op("=").Id("c").Dot("handleRequest").Call(handleRequestParams).Op(";").Id("e").Op("!=").Nil().BlockFunc(requestErrorBlock)

							if m.Reply {
								g.Return(Nil())
							}
							return
						}

						g.If().Id("d").Op(",").Id("e").Op("=").Id("c").Dot("handleRequest").Call(handleRequestParams).Op(";").Id("e").Op("!=").Nil().BlockFunc(requestErrorBlock).Else().If().Id("e").Op("=").Qual("encoding/json", "Unmarshal").CallFunc(func(g *Group) {
							g.Id("d")

							if !m.Results[0].Pointer || m.Results[0].Array {
								g.Add(Op("&").Id("r"))
							} else {
								g.Id("r")
							}
						}).Op(";").Id("e").Op("!=").Nil().Block(
							Return(Nil(), Qual("fmt", "Errorf").Call(
								Lit("unable to unmarshal response: %s"), Id("e").Dot("Error").Call()),
							),
						).Else().Block(
							Return(Id("r"), Nil()),
						)
					})

				}

				t.Func().Params(Id("c").Op("*").Id(clientName)).Id("handleRequest").Params(Id("r").Op("*").Qual("github.com/zyra/autonats", "Request")).Params(Op("[]").Byte(), Error()).Block(
					If(Id("r").Op("==").Nil()).Block(
						Return(Nil(), Qual("errors", "New").Call(Lit("request is nil"))),
					).Else().If().Id("d").Op(",").Id("e").Op(":=").Qual("encoding/json", "Marshal").Call(Id("r")).Op(";").Id("e").Op("!=").Nil().Block(
						Return(Nil(), Qual("fmt", "Errorf").Call(Lit("unable to marshal request: %s"), Id("e").Dot("Error").Call())),
					).Else().If().Id("m").Op(",").Id("e").Op(":=").Id("c").Dot("nc").Dot("Request").Call(Id("r").Dot("Subject"), Id("d"), Qual("time", "Second").Op("*").Lit(p.NatsTimeout)).Op(";").Id("e").Op("!=").Nil().Block(
						Return(Nil(), Id("e")),
					).Else().If().Id("e").Op(":=").Qual("encoding/json", "Unmarshal").Call(Id("m").Dot("Data"), Id("r")).Op(";").Id("e").Op("!=").Nil().Block(
						Return(Nil(), Id("e")),
					).Else().If().Id("r").Dot("Error").Op("!=").Nil().Block(
						Return(Nil(), Id("r").Dot("Error")),
					).Else().Block(
						Return(Id("r").Dot("Data"), Nil()),
					),
				)

				t.Func().Id("New"+clientName).Params(
					Id("nc").Op("*").Qual("github.com/nats-io/nats.go", "Conn"),
					Id("l").Qual("github.com/zyra/autonats", "Logger"),
				).Params(
					Op("*").Id(clientName),
				).Block(
					Id("c").Op(":=").New(Id(clientName)),
					Id("c").Dot("nc").Op("=").Id("nc"),
					If(Id("l").Op("==").Nil()).Block(
						Id("l").Op("=").Qual("log", "New").Call(
							Qual("os", "Stdout"),
							Lit("[Nats][UserClient]"),
							Qual("log", "LstdFlags"),
						),
					),
					Id("c").Dot("l").Op("=").Id("l"),
					Return(Id("c")),
				)
			}

			{
				t.Type().Id(handlerName).Struct(
					Id("server").Id(s.InterfaceID),
					Id("sub").Op("*").Qual("github.com/nats-io/nats.go", "Subscription"),
				)

				t.Func().Id("New"+handlerName).Params(
					Id("ctx").Qual("context", "Context"),
					Id("server").Id(s.InterfaceID),
					Id("nc").Op("*").Qual("github.com/nats-io/nats.go", "Conn"),
				).Params(
					Op("*").Id(handlerName),
					Error(),
				).Block(
					Id("handler").Op(":=").New(Id(handlerName)),
					Id("handler").Dot("server").Op("=").Id("server"),
					Id("ch").Op(":=").Make(Chan().Op("*").Qual("github.com/nats-io/nats.go", "Msg"), Lit(10)),
					If().Id("s").Op(",").Id("e").Op(":=").Id("nc").Dot("ChanSubscribe").Call(
						Lit("gonats."+s.Name+".>"),
						Id("ch"),
					).Op(";").Id("e").Op("!=").Nil().Block(
						Return(Nil(), Id("e")),
					).Else().Block(
						Id("handler").Dot("sub").Op("=").Id("s"),
						Go().Id("handler").Dot("start").Call(Id("ctx"), Id("ch")),
						Return(Id("handler"), Nil()),
					),
				)

				t.Func().Params(
					Id("s").Op("*").Id(handlerName),
				).Id("start").Params(
					Id("ctx").Qual("context", "Context"),
					Id("ch").Op("<-").Chan().Op("*").Qual("github.com/nats-io/nats.go", "Msg"),
				).Block(
					For().Block(
						Select().Block(
							Case(Op("<-").Id("ctx").Dot("Done").Call()).Block(Return()),
							Case(Id("msg").Op(":=").Op("<-").Id("ch")).Block(Id("s").Dot("handleServerRequest").Call(Id("msg"))),
						),
					),
				)

				t.Func().Params(
					Id("s").Op("*").Id(handlerName),
				).Id("handleServerRequest").Params(
					Id("msg").Op("*").Qual("github.com/nats-io/nats.go", "Msg"),
				).Block(
					If(Id("msg").Op("==").Nil()).Block(
						Return(),
					),
					Var().Id("r").Qual("github.com/zyra/autonats", "Request"),
					Var().Id("err").Error(),
					If().Id("err").Op("=").Qual("encoding/json", "Unmarshal").Call(Id("msg").Dot("Data"), Op("&").Id("r")).Op(";").Id("err").Op("!=").Nil().Block(
						Return(),
					),
					Switch(Id("msg").Dot("Subject")).BlockFunc(func(g *Group) {
						for _, m := range s.Methods {
							subject := "gonats." + s.Name + "." + m.Name
							it := g.Case(Lit(subject))

							if len(m.Params) > 0 {
								it = g.Var().Id("p")

								if m.Params[0].Array {
									it.Op("[]")

									if m.Params[0].Pointer {
										it.Op("*")
									}
								}

								if m.Params[0].TypePackage != "" {
									it.Qual(imps[m.Params[0].TypePackage], m.Params[0].Type)
								} else {
									it.Id(m.Params[0].Type)
								}

								it = g.If().Id("err").Op("=").Qual("encoding/json", "Unmarshal").Call(Id("r").Dot("Data"), Op("&").Id("p")).Op(";").Id("err").Op("!=").Nil().Block(
									Return(),
								)
							}

							it = g.Id("r").Dot("Data").Op("=").Nil()

							if !m.Reply {
								it = g.Id("s").Dot("server").Dot(m.Name)

								if len(m.Params) > 0 {
									if m.Params[0].Pointer && !m.Params[0].Array {
										it.Call(Op("&").Id("p"))
									} else {
										it.Call(Id("p"))
									}
								} else {
									it.Call()
								}

								it = g.If().Id("err").Op("=").Id("msg").Dot("Respond").Call(Make(Op("[]").Byte(), Lit(0))).Op(";").Id("err").Op("!=").Nil().Block(
									Qual("fmt", "Printf").Call(Lit("unable to send response: %s\n"), Id("err").Dot("Error").Call()),
								)

								it = g.Return()
								continue
							}

							it = g.If()

							if len(m.Results) > 1 {
								it.Id("result").Op(",")
							}

							it.Id("err").Op(":=").Id("s").Dot("server").Dot(m.Name)

							if len(m.Params) > 0 {
								if m.Params[0].Pointer && !m.Params[0].Array {
									it.Call(Op("&").Id("p"))
								} else {
									it.Call(Id("p"))
								}
							} else {
								it.Call()
							}

							it.Op(";").Id("err").Op("!=").Nil().Block(
								Id("r").Dot("Error").Op("=").Id("err"),
							)

							if len(m.Results) > 1 {
								it.Else().If().Id("b").Op(",").Id("err").Op(":=").Qual("encoding/json", "Marshal")

								if !m.Results[0].Pointer || m.Results[0].Array {
									it.Call(Op("&").Id("result"))
								} else {
									it.Call(Id("result"))
								}

								it.Op(";").Id("err").Op("!=").Nil().Block(
									Qual("fmt", "Printf").Call(Lit("unable to marshal response: %s\n"), Id("err").Dot("Error").Call()),
									Return(),
								)

								it.Else().Block(
									Id("r").Dot("Data").Op("=").Id("b"),
								)
							}
						}

						g.Default().Block(Return())
					}),
					If().Id("b").Op(",").Id("err").Op(":=").Qual("encoding/json", "Marshal").Call(Id("r")).Op(";").Id("err").Op("!=").Nil().Block(
						Qual("fmt", "Printf").Call(Lit("unable to marshal response object: %s\n"), Id("err").Dot("Error").Call()),
						Return(),
					).Else().If().Id("err").Op("=").Id("msg").Dot("Respond").Call(Id("b")).Op(";").Id("err").Op("!=").Nil().Block(
						Qual("fmt", "Printf").Call(Lit("unable to send response: %s\n"), Id("err").Dot("Error").Call()),
					),
				)
			}
		}

		buff := bytes.NewBuffer(make([]byte, 0))

		if err := t.Render(buff); err != nil {
			str := err.Error()
			panic(str)
		}

		if err := ioutil.WriteFile(filepath.Join(pkg.BaseDir, p.OutFileName), buff.Bytes(), 0655); err != nil {
			panic(err)
		}
	}
}
