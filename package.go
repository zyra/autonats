package autonats

type Package struct {
	Services       []*Service
	Imports        map[string]string
	Name           string
	BaseDir        string
	OriginFileName string
	FileName       string
}

func PackageFromService(svc *Service) *Package {
	return &Package{
		Services:       make([]*Service, 0),
		Imports:        make(map[string]string),
		Name:           svc.PackageName,
		BaseDir:        svc.Basedir,
		OriginFileName: svc.FileName,
		FileName:       svc.FileName,
	}
}

func (pkg *Package) AddService(service *Service) {
	pkg.Services = append(pkg.Services, service)

	for ik, iv := range service.Imports {
		pkg.Imports[ik] = iv
	}
}
