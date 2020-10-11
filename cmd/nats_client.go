package cmd

import (
	"context"
	"github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/not.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/zyra/autonats"
	"github.com/zyra/autonats/example"
	"time"
)

const timeout = time.Second * 5

type ImageServer interface {
	GetByUserId(ctx context.Context, userId string) ([]*example.Image, error)
	GetCountByUserId(ctx context.Context, userId string) (int, error)
}

type imageHandler struct {
	Server  ImageServer
	nc      *nats.Conn
	runners []*autonats.Runner
}

func (h *imageHandler) Run(ctx context.Context) error {
	h.runners = make([]*autonats.Runner, 2, 2)
	tracer := opentracing.GlobalTracer()

	if runner, err := autonats.StartRunner(ctx, h.nc, "autonats.Image.GetByUserId", "autonats", 5, func(msg *nats.Msg) {
		t := not.NewTraceMsg(msg)
		// Extract the span context from the request message.
		sc, err := tracer.Extract(opentracing.Binary, t)
		if err != nil {
			return
		}
		replySpan := tracer.StartSpan("Autonats ImageServer", ext.SpanKindRPCServer, ext.RPCServerOption(sc))
		ext.MessageBusDestination.Set(replySpan, msg.Subject)
		defer replySpan.Finish()
		innerCtx, _ := context.WithTimeout(ctx, timeout)
		innerCtxT := opentracing.ContextWithSpan(innerCtx, replySpan)

		result, err := h.Server.GetByUserId(innerCtxT, string(msg.Data))

		reply := autonats.GetReply()
		defer autonats.PutReply(reply)

		if err != nil {
			replySpan.LogFields(log.Event("Handler returned error"))
			reply.Error = []byte(err.Error())
		} else if result != nil {
			replySpan.LogFields(log.Event("Handler returned a result"))
			if err := reply.MarshalAndSetData(result); err != nil {
				replySpan.LogFields(log.Error(err))
				return
			}
		}

		replyData, err := reply.MarshalBinary()

		if err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}

		replySpan.LogFields(log.Event("Sending reply"))
		if err := msg.Respond(replyData); err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}
	}); err != nil {
		return err
	} else {
		h.runners[0] = runner
	}

	if runner, err := autonats.StartRunner(ctx, h.nc, "autonats.Image.GetCountByUserId", "autonats", 5, func(msg *nats.Msg) (interface{}, error) {
		innerCtx, _ := context.WithTimeout(ctx, time.Second*5)
		return h.Server.GetCountByUserId(innerCtx, string(msg.Data))
	}); err != nil {
		h.Shutdown()
		return err
	} else {
		h.runners[1] = runner
	}

	return nil
}

func (h *imageHandler) Shutdown() {
	for i := range h.runners {
		if h.runners[i] != nil {
			_ = h.runners[i].Shutdown()
		}
	}
}

func NewImageHandler(server ImageServer, nc *nats.Conn) autonats.Handler {
	return &imageHandler{
		Server: server,
		nc:     nc,
	}
}

type ImageClient struct {
	nc  *nats.Conn
	log autonats.Logger
}

func (client *ImageClient) GetByUserId(ctx context.Context, userId string) ([]*example.Image, error) {
	reqSpan, reqCtx := opentracing.StartSpanFromContext(ctx, "Autonats ImageClient Request", ext.SpanKindRPCClient)
	ext.MessageBusDestination.Set(reqSpan, "autonats.Image.GetByUserId")
	defer reqSpan.Finish()
	reqSpan.LogFields(log.Event("Starting request"))

	var t not.TraceMsg

	if err := opentracing.GlobalTracer().Inject(reqSpan.Context(), opentracing.Binary, &t); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	if _, err := t.Write([]byte(userId)); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	reqCtx, cancelFn := context.WithTimeout(reqCtx, timeout)
	defer cancelFn()

	if replyMsg, err := client.nc.RequestWithContext(ctx, "autonats.Image.GetByUserId", t.Bytes()); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	} else {
		reqSpan.LogFields(log.Event("Reply received"))
		reply := autonats.GetReply()
		defer autonats.PutReply(reply)
		if err := reply.UnmarshalBinary(replyMsg.Data); err != nil {
			reqSpan.LogFields(log.Error(err))
			return nil, err
		}

		var result []*example.Image
		reply.GetDataAsString()
		reply.UnmarshalData(&result)

		if err := reply.GetError(); err != nil {
			return nil, err
		} else if err := reply.UnmarshalData(&result); err != nil {
			return nil, err
		} else {
			return result, nil
		}
	}
}

func (client *ImageClient) GetCountByUserId(ctx context.Context, userId string) (int, error) {
	var dest int

	if err := autonats.SendRequest(ctx, client.nc, "autonats.Image.GetCountByUserId", userId, &dest); err != nil {
		return 0, err
	} else {
		return dest, nil
	}
}

type UserServer interface {
	GetById(ctx context.Context, id []byte) (*example.User, error)
	Create(ctx context.Context, user *example.User) error
}

type userHandler struct {
	Server  UserServer
	nc      *nats.Conn
	runners []*autonats.Runner
}

func (h *userHandler) Run(ctx context.Context) error {
	h.runners = make([]*autonats.Runner, 2, 2)
	if runner, err := autonats.StartRunner(ctx, h.nc, "autonats.User.GetById", "autonats", 5, func(msg *nats.Msg) (interface{}, error) {
		var data []byte
		if err := jsoniter.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		} else {
			innerCtx, _ := context.WithTimeout(ctx, time.Second*5)
			return h.Server.GetById(innerCtx, data)
		}
	}); err != nil {
		return err
	} else {
		h.runners[0] = runner
	}

	if runner, err := autonats.StartRunner(ctx, h.nc, "autonats.User.Create", "autonats", 5, func(msg *nats.Msg) (interface{}, error) {
		var data example.User
		if err := jsoniter.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		} else {
			innerCtx, _ := context.WithTimeout(ctx, time.Second*5)
			return nil, h.Server.Create(innerCtx, &data)
		}
	}); err != nil {
		h.Shutdown()
		return err
	} else {
		h.runners[1] = runner
	}

	return nil
}

func (h *userHandler) Shutdown() {
	for i := range h.runners {
		if h.runners[i] != nil {
			_ = h.runners[i].Shutdown()
		}
	}
}

func NewUserHandler(server UserServer, nc *nats.Conn) autonats.Handler {
	return &userHandler{
		Server: server,
		nc:     nc,
	}
}

type UserClient struct {
	nc  *nats.EncodedConn
	log autonats.Logger
}

func (client *UserClient) GetById(ctx context.Context, id []byte) (*example.User, error) {
	var dest example.User

	if err := autonats.SendRequest(ctx, client.nc, "autonats.User.GetById", id, &dest); err != nil {
		return nil, err
	} else {
		return &dest, nil
	}
}

func (client *UserClient) Create(ctx context.Context, user *example.User) error {
	return autonats.SendRequest(ctx, client.nc, "autonats.User.Create", user, nil)
}
