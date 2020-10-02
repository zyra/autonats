package autonats

import (
	"github.com/nats-io/nats.go"
	"time"
)
import "context"

func SendRequest(ctx context.Context, nc *nats.EncodedConn, subject string, in, out interface{}) error {
	reqCtx, cancelFn := context.WithTimeout(ctx, time.Second*3)
	defer cancelFn()
	return nc.RequestWithContext(reqCtx, subject, in, out)
}
