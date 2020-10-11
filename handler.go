package autonats

import (
	"context"
	"github.com/nats-io/nats.go"
)

type Handler interface {
	Run(ctx context.Context) error // Subscribes to the queue and dispatches handler go-routines
	Shutdown()                     // Shuts down all handlers gracefully
}

type Runner struct {
	sub *nats.Subscription
}

func (r *Runner) Shutdown() error {
	return r.sub.Unsubscribe()
}

func StartRunner(ctx context.Context, nc *nats.Conn, subj, group string, concurrency int, handleFn func(msg *nats.Msg)) (*Runner, error) {
	subChan := make(chan *nats.Msg)

	sub, err := nc.ChanQueueSubscribe(subj, group, subChan)

	if err != nil {
		return nil, err
	}

	for i := 0; i < concurrency; i++ {
		go func() {
			var msg *nats.Msg
			var ok bool

			for {
				select {
				case <-ctx.Done():
					return

				case msg, ok = <-subChan:
					if !ok {
						return
					}

					handleFn(msg)
				}
			}
		}()
	}

	return &Runner{sub: sub}, nil
}
