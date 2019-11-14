package autonats

// Logger for service client
type Logger interface {
	Printf(format string, v ...interface{})
}

// Request object that is passed between server handler and client
type Request struct {
	Subject string `json:"-"`           // nats subject, used for processing only and not sent via nats
	Data    []byte `json:"d,omitempty"` // params (on server) or result (on client)
	Error   error  `json:"e,omitempty"` // error sent from server handler
}
