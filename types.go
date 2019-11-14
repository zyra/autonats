package autonats

type Logger interface {
	Printf(format string, v ...interface{})
}

type Request struct {
	Subject string `json:"-"`
	Data    []byte `json:"d,omitempty"`
	Error   error  `json:"e,omitempty"`
}