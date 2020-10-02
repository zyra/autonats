package autonats

// Logger for Service client
type Logger interface {
	Printf(format string, v ...interface{})
}
