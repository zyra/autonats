package autonats

import (
	"context"
	"encoding/json"
	"fmt"
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

func StartRunner(ctx context.Context, nc *nats.Conn, subj, group string, concurrency int, handleFn func(msg *nats.Msg) (interface{}, error)) (*Runner, error) {
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

					res, err := handleFn(msg)

					var reply Reply

					if err != nil {
						reply.Error = []byte(err.Error())
					} else if res != nil {
						if data, err := json.Marshal(res); err != nil {
							reply.Error = []byte(fmt.Sprintf("failed to marshal response: %s", err.Error()))
						} else {
							reply.Data = data
						}
					}

					if data, err := json.Marshal(&reply); err != nil {
						// TODO handle this eventually
					} else if err := msg.Respond(data); err != nil {
						// TODO handle this eventually
					}
				}
			}
		}()
	}

	return &Runner{sub: sub}, nil
}
